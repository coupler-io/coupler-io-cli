package datasets

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var dataflowID, query, name string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Add a custom SQL dataset to a dataflow",
		Long: "Creates a dataset defined by a SELECT statement over the dataflow's source tables. " +
			"Quote table names. Use `coupler-io datasets list --dataflow-id <id>` and `datasets get` to find them.",
		Example: `  coupler-io datasets create --dataflow-id <id> --name "Summary" \
      --query 'select country, sum(revenue) as revenue from "sales" group by country'
  coupler-io datasets create --dataflow-id <id> --query @summary.sql`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireFlag("dataflow-id", dataflowID); err != nil {
				return err
			}
			if err := cmdutil.RequireFlag("query", query); err != nil {
				return err
			}
			sql, err := f.ReadArg(query)
			if err != nil {
				return err
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows", dataflowID, "datasets")
			if err != nil {
				return err
			}

			body := map[string]interface{}{"type": "customSql", "query": sql}
			if name != "" {
				body["name"] = name
			}

			var result json.RawMessage
			if err := c.Post(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, fmt.Sprintf("Dataset created in dataflow %s.", dataflowID), detailColumns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&dataflowID, "dataflow-id", "", "Dataflow to add the dataset to (required)")
	flags.StringVar(&query, "query", "", "SELECT statement, @file or - for stdin (required)")
	flags.StringVar(&name, "name", "", "Dataset name (default is generated)")
	return cmd
}
