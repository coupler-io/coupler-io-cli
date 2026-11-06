package dataflows

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List dataflows in the workspace",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows")
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, nil, &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{}, listColumns)
		},
	}
}
