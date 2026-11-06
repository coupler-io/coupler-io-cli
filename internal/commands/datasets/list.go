package datasets

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var fl filters

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List datasets in the workspace",
		Long: "Lists datasets with their state. Only datasets with QUERYABLE=true can be used with " +
			"`datasets query`. Use --output json to see the state legend.",
		Example: `  coupler datasets list --dataflow-id <id>
  coupler datasets list --source gsheet --query sales`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("datasets")
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, fl.values(), &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{Items: "datasets", Footers: []output.Footer{totalFooter}}, listColumns)
		},
	}

	fl.bind(cmd)
	return cmd
}
