package dataflows

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show a dataflow and the state of its sources",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows", args[0])
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, nil, &body); err != nil {
				return err
			}
			return f.PrintDetail(body, detailColumns,
				cmdutil.Section{Title: "Sources", Key: "sources", Columns: sourceColumns})
		},
	}
}

func newCmdSetup(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "setup <id>",
		Short: "Show the full setup of a dataflow (schedule, sources, destinations, views)",
		Long: "Returns the complete configuration, including every source and destination with its params " +
			"and what is still missing. Always printed as JSON.",
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows", args[0], "setup")
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, nil, &body); err != nil {
				return err
			}
			return f.PrintJSON(body)
		},
	}
}
