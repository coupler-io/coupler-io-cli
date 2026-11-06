package datasets

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newCmdGet(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show a dataset and its columns",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("datasets", args[0])
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, nil, &body); err != nil {
				return err
			}
			return f.PrintDetail(body, detailColumns,
				cmdutil.Section{Title: "Columns", Key: "columns", Columns: columnColumns})
		},
	}
}
