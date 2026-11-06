package datasets

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

func newCmdQuery(f *cmdutil.Factory) *cobra.Command {
	var statement string

	cmd := &cobra.Command{
		Use:   "query <id>",
		Short: "Run a DuckDB SQL SELECT against a dataset's latest data",
		Long: `Runs a SELECT statement on the latest successful run of a dataset. Read the rows from the
table named "data". Use "datasets get <id>" to see the column names (DB_NAME).

This is for analysis (aggregates, filters, samples), not exports: at most 10,000 rows are
returned and the request times out after 60 seconds. To export all data, add a destination
to the dataflow instead.

Use --output csv for spreadsheet-friendly results or --output json for the raw response.`,
		Example: `  coupler datasets query <id> --statement 'select * from data limit 5'
  coupler datasets query <id> -s @report.sql -o csv > report.csv
  echo 'select count(*) from data' | coupler datasets query <id> -s -`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireFlag("statement", statement); err != nil {
				return err
			}
			sql, err := f.ReadArg(statement)
			if err != nil {
				return err
			}
			if sql == "" {
				return cmdutil.UsageErrorf("--statement must not be empty")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("datasets", args[0], "query")
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Post(cmd.Context(), path, map[string]string{"statement": sql}, &body); err != nil {
				return err
			}

			var result struct {
				Rows      json.RawMessage `json:"rows"`
				Truncated bool            `json:"truncated"`
			}
			if err := json.Unmarshal(body, &result); err != nil {
				return fmt.Errorf("unmarshal response: %w", err)
			}

			format := f.Format()
			if format != output.FormatJSON {
				columns, err := output.RowColumns(result.Rows)
				if err != nil {
					return err
				}
				if len(columns) == 0 {
					if format != output.FormatCSV {
						fmt.Fprintln(f.IOStreams.Out, "(no rows)")
					}
				} else if err := output.Print(f.IOStreams.Out, format, result.Rows, columns); err != nil {
					return err
				}
			} else if err := f.PrintJSON(body); err != nil {
				return err
			}

			if result.Truncated {
				fmt.Fprintln(f.IOStreams.ErrOut,
					"Warning: result truncated to 10,000 rows. Aggregate, filter or add LIMIT for a complete result.")
			}
			return nil
		},
	}

	cmd.Flags().StringVarP(&statement, "statement", "s", "", "SELECT statement in DuckDB SQL, @file or - for stdin (required)")
	return cmd
}
