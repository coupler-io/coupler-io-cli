package datasets

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var name, aiContext, query string

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Rename a dataset, edit its AI context, or change a custom SQL query",
		Long: "--ai-context stores Markdown that describes the dataset for AI agents. " +
			"--query only works on custom SQL datasets. Pass an empty string to --name or --ai-context to clear it. " +
			"Values may be @file or - (stdin).",
		Example: `  coupler datasets update <id> --name "Campaigns 2026"
  coupler datasets update <id> --ai-context @notes.md
  coupler datasets update <id> --query 'select id from "campaigns"'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			body := map[string]interface{}{}

			if flags.Changed("name") {
				body["name"] = nullIfEmpty(name)
			}
			if flags.Changed("ai-context") {
				text, err := f.ReadArg(aiContext)
				if err != nil {
					return err
				}
				body["ai_context"] = nullIfEmpty(text)
			}
			if flags.Changed("query") {
				sql, err := f.ReadArg(query)
				if err != nil {
					return err
				}
				body["type"] = "customSql"
				body["query"] = sql
			}
			if len(body) == 0 {
				return cmdutil.UsageErrorf("nothing to update: set --name, --ai-context or --query")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("datasets", args[0])
			if err != nil {
				return err
			}

			var result json.RawMessage
			if err := c.Patch(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, fmt.Sprintf("Dataset %s updated.", args[0]), detailColumns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&name, "name", "", "Dataset name (empty clears it)")
	flags.StringVar(&aiContext, "ai-context", "", "Markdown description for AI agents, @file or - (empty clears it)")
	flags.StringVar(&query, "query", "", "New SELECT statement for a custom SQL dataset, @file or -")
	return cmd
}

// nullIfEmpty maps "" to JSON null, which the API uses to clear a value.
func nullIfEmpty(s string) interface{} {
	if s == "" {
		return nil
	}
	return s
}
