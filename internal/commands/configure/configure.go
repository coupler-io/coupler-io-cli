package configure

import (
	"bufio"
	"encoding/json"
	"fmt"
	"io"
	"os"
	"strconv"
	"strings"

	"github.com/spf13/cobra"
	"gopkg.in/yaml.v3"

	"github.com/coupler-io/coupler-io-cli/internal/client"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/config"
)

type configFile struct {
	APIToken    string `yaml:"api-token"`
	WorkspaceID string `yaml:"workspace-id"`
	APIURL      string `yaml:"api-url,omitempty"`
}

type workspace struct {
	ID           json.Number `json:"id"`
	Name         string      `json:"name"`
	Organization string      `json:"organization"`
}

func NewCmdConfigure(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:   "configure",
		Short: "Save your API token and default workspace",
		Long: `Validates your personal API token, picks a workspace, and saves both to ~/.coupler-io.yaml
(readable only by you).

Get a token in the API access section of your Coupler.io profile: https://app.coupler.io/app/profile

The token comes from --api-token-stdin, COUPLER_IO_API_TOKEN or --api-token (the last one is visible
in your shell history and process list). If your token can access several workspaces
and you do not pass --workspace-id, you are asked to choose one.`,
		Example: `  echo "$COUPLER_IO_TOKEN" | coupler-io configure --api-token-stdin --workspace-id 42
  coupler-io configure --api-token <token>`,
		Args: cobra.NoArgs,
		// Must run even when the existing config file is corrupt: this
		// command is how the user overwrites it.
		Annotations: map[string]string{cmdutil.AnnotationIgnoreConfigError: "true"},
		RunE: func(cmd *cobra.Command, args []string) error {
			token, err := config.RequireAPIToken(f.Viper)
			if err != nil {
				return err
			}

			apiURL := f.Viper.GetString(config.KeyAPIURL)
			c := f.ClientOverride
			if c == nil {
				c = client.New(token, client.WithBaseURL(apiURL))
			}

			var resp struct {
				Workspaces []workspace `json:"workspaces"`
			}
			if err := c.Get(cmd.Context(), cmdutil.Path("workspaces"), nil, &resp); err != nil {
				return fmt.Errorf("could not validate the API token: %w", err)
			}
			if len(resp.Workspaces) == 0 {
				return fmt.Errorf("this API token has no accessible workspaces")
			}

			ws, err := chooseWorkspace(f, resp.Workspaces, f.Viper.GetString(config.KeyWorkspaceID))
			if err != nil {
				return err
			}

			cfg := configFile{APIToken: token, WorkspaceID: ws.ID.String()}
			if apiURL != "" && apiURL != client.DefaultBaseURL {
				cfg.APIURL = apiURL
			}
			out, err := yaml.Marshal(&cfg)
			if err != nil {
				return fmt.Errorf("marshal config: %w", err)
			}

			path, err := config.ConfigFilePath()
			if err != nil {
				return err
			}
			if err := os.WriteFile(path, out, 0o600); err != nil {
				return fmt.Errorf("write config file: %w", err)
			}
			// WriteFile keeps the mode of an existing file, so tighten it explicitly.
			if err := os.Chmod(path, 0o600); err != nil {
				return fmt.Errorf("restrict config file permissions: %w", err)
			}

			fmt.Fprintf(f.IOStreams.Out, "Authenticated. Workspace: %s (id %s)\n", ws.Name, ws.ID)
			fmt.Fprintf(f.IOStreams.Out, "Configuration saved to %s\n", path)
			return nil
		},
	}

	return cmd
}

func chooseWorkspace(f *cmdutil.Factory, list []workspace, wanted string) (workspace, error) {
	if wanted != "" {
		for _, ws := range list {
			if ws.ID.String() == wanted {
				return ws, nil
			}
		}
		return workspace{}, fmt.Errorf("workspace %s is not accessible with this token; available:\n%s", wanted, describe(list))
	}
	if len(list) == 1 {
		return list[0], nil
	}

	fmt.Fprintf(f.IOStreams.Out, "Workspaces:\n%s\n", describe(list))
	fmt.Fprintf(f.IOStreams.Out, "Select a workspace [1-%d]: ", len(list))

	line, err := bufio.NewReader(f.IOStreams.In).ReadString('\n')
	if err != nil && (err != io.EOF || strings.TrimSpace(line) == "") {
		return workspace{}, fmt.Errorf("multiple workspaces found; pass --workspace-id <id>")
	}
	n, convErr := strconv.Atoi(strings.TrimSpace(line))
	if convErr != nil || n < 1 || n > len(list) {
		return workspace{}, fmt.Errorf("invalid selection %q", strings.TrimSpace(line))
	}
	return list[n-1], nil
}

func describe(list []workspace) string {
	lines := make([]string, len(list))
	for i, ws := range list {
		lines[i] = fmt.Sprintf("  %d) %s (%s) - id %s", i+1, ws.Name, ws.Organization, ws.ID)
	}
	return strings.Join(lines, "\n")
}
