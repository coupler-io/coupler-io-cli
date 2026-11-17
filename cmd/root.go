package cmd

import (
	"context"
	"fmt"
	"io"
	"os"
	"os/signal"
	"strings"
	"sync/atomic"
	"syscall"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/client"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/commands/configure"
	"github.com/coupler-io/coupler-io-cli/internal/commands/credentials"
	"github.com/coupler-io/coupler-io-cli/internal/commands/dataflows"
	"github.com/coupler-io/coupler-io-cli/internal/commands/datasets"
	"github.com/coupler-io/coupler-io-cli/internal/commands/destinations"
	"github.com/coupler-io/coupler-io-cli/internal/commands/integrations"
	"github.com/coupler-io/coupler-io-cli/internal/commands/skills"
	"github.com/coupler-io/coupler-io-cli/internal/commands/sources"
	"github.com/coupler-io/coupler-io-cli/internal/commands/templates"
	"github.com/coupler-io/coupler-io-cli/internal/commands/workspaces"
	"github.com/coupler-io/coupler-io-cli/internal/config"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

func NewRootCmd(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "coupler-io",
		Short: "CLI for the Coupler.io data integration platform",
		Long: `A command-line interface for Coupler.io: manage dataflows, sources and destinations,
and query the data they produce.

Get started:
  echo "$COUPLER_IO_TOKEN" | coupler-io configure --api-token-stdin
  coupler-io dataflows list`,
		SilenceUsage:  true,
		SilenceErrors: true,
		PersistentPreRunE: func(cmd *cobra.Command, args []string) error {
			if f.ConfigErr != nil && !cmd.HasSubCommands() {
				if _, ok := cmd.Annotations[cmdutil.AnnotationIgnoreConfigError]; !ok {
					return f.ConfigErr
				}
			}
			if err := readTokenFromStdin(cmd, f); err != nil {
				return err
			}
			if _, err := output.ParseFormat(f.Viper.GetString(config.KeyOutput)); err != nil {
				return cmdutil.WrapUsage(err)
			}
			return nil
		},
	}
	cmd.SetFlagErrorFunc(func(_ *cobra.Command, err error) error {
		return cmdutil.WrapUsage(err)
	})

	flags := cmd.PersistentFlags()
	flags.String("api-token", "", "Coupler.io API token; visible in shell history and the process list, prefer --api-token-stdin or COUPLER_IO_API_TOKEN")
	flags.Bool("api-token-stdin", false, "Read the API token from standard input")
	flags.StringP("workspace-id", "w", "", "Workspace id (env: COUPLER_IO_WORKSPACE_ID)")
	flags.String("api-url", "", "API base URL, default "+client.DefaultBaseURL+" (env: COUPLER_IO_API_URL)")
	flags.StringP("output", "o", "table", "Output format: table, json, text, csv (env: COUPLER_IO_OUTPUT)")

	for _, name := range []string{"api-token", "workspace-id", "api-url", "output"} {
		_ = f.Viper.BindPFlag(name, flags.Lookup(name))
	}

	_ = cmd.RegisterFlagCompletionFunc("output", func(*cobra.Command, []string, string) ([]string, cobra.ShellCompDirective) {
		names := make([]string, len(output.Formats))
		for i, format := range output.Formats {
			names[i] = string(format)
		}
		return names, cobra.ShellCompDirectiveNoFileComp
	})

	// Setup
	cmd.AddCommand(configure.NewCmdConfigure(f))
	cmd.AddCommand(workspaces.NewCmdWorkspaces(f))

	// Dataflows
	cmd.AddCommand(dataflows.NewCmdDataflows(f))
	cmd.AddCommand(sources.NewCmdSources(f))
	cmd.AddCommand(destinations.NewCmdDestinations(f))

	// Data
	cmd.AddCommand(datasets.NewCmdDatasets(f))

	// Discovery
	cmd.AddCommand(integrations.NewCmdIntegrations(f))
	cmd.AddCommand(credentials.NewCmdCredentials(f))
	cmd.AddCommand(templates.NewCmdTemplates(f))
	cmd.AddCommand(skills.NewCmdSkills(f))

	cmd.AddCommand(newCompletionCmd())

	finalizeTree(cmd)
	return cmd
}

// readTokenFromStdin implements --api-token-stdin: the token never appears in
// argv, shell history or the process list.
func readTokenFromStdin(cmd *cobra.Command, f *cmdutil.Factory) error {
	fromStdin, err := cmd.Flags().GetBool("api-token-stdin")
	if err != nil || !fromStdin {
		return nil
	}
	if flag := cmd.Flag("api-token"); flag != nil && flag.Changed {
		return cmdutil.UsageErrorf("--api-token and --api-token-stdin are mutually exclusive")
	}
	if f.StdinUsedBy != "" {
		return cmdutil.UsageErrorf("standard input is already used by %s", f.StdinUsedBy)
	}
	f.StdinUsedBy = "--api-token-stdin"
	data, err := io.ReadAll(f.IOStreams.In)
	if err != nil {
		return fmt.Errorf("read API token from stdin: %w", err)
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return cmdutil.UsageErrorf("--api-token-stdin: standard input is empty")
	}
	f.Viper.Set(config.KeyAPIToken, token)
	return nil
}

// finalizeTree makes cobra's argument errors exit with the usage code (2):
//
//   - Command groups (commands that only have subcommands) become runnable so
//     that an unknown subcommand is a usage error rather than silently
//     printing help and exiting 0. With no arguments they still print help.
//   - Every positional-argument validator is wrapped to return a UsageError.
func finalizeTree(c *cobra.Command) {
	if c.HasSubCommands() && c.Run == nil && c.RunE == nil {
		if c.Args == nil {
			c.Args = cobra.ArbitraryArgs
		}
		c.RunE = groupRunE
	}
	if validate := c.Args; validate != nil {
		c.Args = func(cmd *cobra.Command, args []string) error {
			return cmdutil.WrapUsage(validate(cmd, args))
		}
	}
	for _, sub := range c.Commands() {
		finalizeTree(sub)
	}
}

func groupRunE(cmd *cobra.Command, args []string) error {
	if len(args) == 0 {
		return cmd.Help()
	}
	msg := fmt.Sprintf("unknown command %q for %q", args[0], cmd.CommandPath())
	if suggestions := cmd.SuggestionsFor(args[0]); len(suggestions) > 0 {
		msg += "\n\nDid you mean this?\n\t" + strings.Join(suggestions, "\n\t")
	}
	return cmdutil.UsageErrorf("%s", msg)
}

func newCompletionCmd() *cobra.Command {
	return &cobra.Command{
		Use:       "completion [bash|zsh|fish|powershell]",
		Short:     "Generate shell completion scripts",
		Args:      cobra.MatchAll(cobra.ExactArgs(1), cobra.OnlyValidArgs),
		ValidArgs: []string{"bash", "zsh", "fish", "powershell"},
		// Completion output must not depend on a readable config file.
		Annotations: map[string]string{cmdutil.AnnotationIgnoreConfigError: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			switch args[0] {
			case "bash":
				return cmd.Root().GenBashCompletion(cmd.OutOrStdout())
			case "zsh":
				return cmd.Root().GenZshCompletion(cmd.OutOrStdout())
			case "fish":
				return cmd.Root().GenFishCompletion(cmd.OutOrStdout(), true)
			case "powershell":
				return cmd.Root().GenPowerShellCompletionWithDesc(cmd.OutOrStdout())
			}
			return nil
		},
	}
}

var buildVersion = "dev"

// SetVersion records build metadata injected by GoReleaser via ldflags.
func SetVersion(version, commit, date string) {
	buildVersion = fmt.Sprintf("%s (commit: %s, built: %s)", version, commit, date)
	client.UserAgent = "coupler-io-cli/" + version
}

// Execute runs the CLI and returns the process exit code: 0 on success, 2 for
// usage errors, 128+N when terminated by signal N, and 1 for any other error.
func Execute() int {
	ctx, received, stop := notifyContext()
	defer stop()

	f := cmdutil.NewFactory()
	f.LoadConfig()
	root := NewRootCmd(f)
	root.Version = buildVersion
	err := root.ExecuteContext(ctx)

	return report(f.IOStreams.ErrOut, err, received())
}

// report prints err to w and returns the exit code. sig is the signal that
// interrupted the run, or 0.
func report(w io.Writer, err error, sig syscall.Signal) int {
	if sig != 0 {
		fmt.Fprintln(w, "Interrupted")
		return 128 + int(sig)
	}
	if err != nil {
		fmt.Fprintf(w, "Error: %s\n", err)
	}
	return cmdutil.ExitCode(err)
}

// notifyContext returns a context cancelled by SIGINT or SIGTERM. received
// reports which signal fired (0 if none), so the exit code can be 128+N. After
// the first signal the default behaviour is restored, so a second Ctrl-C
// terminates a command that is slow to wind down.
func notifyContext() (ctx context.Context, received func() syscall.Signal, stop func()) {
	ctx, cancel := context.WithCancel(context.Background())
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, os.Interrupt, syscall.SIGTERM)

	var got atomic.Int32
	go func() {
		select {
		case s := <-ch:
			if sig, ok := s.(syscall.Signal); ok {
				got.Store(int32(sig))
			}
			signal.Stop(ch)
			cancel()
		case <-ctx.Done():
		}
	}()

	received = func() syscall.Signal { return syscall.Signal(got.Load()) }
	stop = func() {
		signal.Stop(ch)
		cancel()
	}
	return ctx, received, stop
}
