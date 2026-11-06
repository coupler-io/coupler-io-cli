package dataflows

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newCmdRun(f *cmdutil.Factory) *cobra.Command {
	var mode string

	cmd := &cobra.Command{
		Use:   "run <id>",
		Short: "Run a dataflow now",
		Long: "Queues a run and returns immediately. A full run imports the data; a preview run only " +
			"fetches a sample. A dataflow that is not fully configured is rejected with the missing setup; " +
			"check it with `coupler dataflows setup <id>`.",
		Example: `  coupler dataflows run <id>
  coupler dataflows run <id> --mode preview`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if mode != "full" && mode != "preview" {
				return cmdutil.UsageErrorf("--mode must be full or preview")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows", args[0], "executions")
			if err != nil {
				return err
			}

			var result json.RawMessage
			if err := c.Post(cmd.Context(), path, map[string]string{"run_mode": mode}, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, fmt.Sprintf("Dataflow %s queued (%s run).", args[0], mode), nil)
		},
	}

	cmd.Flags().StringVar(&mode, "mode", "full", "Run mode: full or preview")
	return cmd
}
