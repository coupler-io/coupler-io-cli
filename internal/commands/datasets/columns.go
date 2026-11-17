package datasets

import (
	"encoding/json"
	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var columnUpdateColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "UPDATED_COLUMNS", Field: "updated_column_keys"},
}

func newCmdColumns(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "columns",
		Aliases: []string{"column"},
		Short:   "Edit dataset column labels and descriptions",
	}
	cmd.AddCommand(newCmdColumnsUpdate(f))
	return cmd
}

func newCmdColumnsUpdate(f *cmdutil.Factory) *cobra.Command {
	var key, label, description, updates string

	cmd := &cobra.Command{
		Use:   "update <dataset-id>",
		Short: "Set column labels and descriptions",
		Long: "Update one column with --key plus --label and/or --description, or several at once with " +
			"--updates (a JSON array, @file or -). Columns are matched by key (see `datasets get`). " +
			"Fields you omit stay unchanged; an empty --label or --description clears it.",
		Example: `  coupler-io datasets columns update <dataset-id> --key revenue --label "Revenue" --description "Net, USD"
  coupler-io datasets columns update <dataset-id> --updates '[{"key":"revenue","label":"Revenue"},{"key":"cost","label":"Cost"}]'`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			var columnUpdates []map[string]interface{}

			switch {
			case updates != "":
				if key != "" || flags.Changed("label") || flags.Changed("description") {
					return cmdutil.UsageErrorf("use either --updates or --key/--label/--description, not both")
				}
				raw, err := f.ReadArg(updates)
				if err != nil {
					return err
				}
				if err := json.Unmarshal([]byte(raw), &columnUpdates); err != nil {
					return cmdutil.UsageErrorf("--updates must be a JSON array of {key,label,description}: %w", err)
				}
			case key != "":
				if !flags.Changed("label") && !flags.Changed("description") {
					return cmdutil.UsageErrorf("set --label and/or --description")
				}
				update := map[string]interface{}{"key": key}
				if flags.Changed("label") {
					update["label"] = nullIfEmpty(label)
				}
				if flags.Changed("description") {
					update["description"] = nullIfEmpty(description)
				}
				columnUpdates = append(columnUpdates, update)
			default:
				return cmdutil.UsageErrorf("--key (with --label/--description) or --updates is required")
			}
			if len(columnUpdates) == 0 {
				return cmdutil.UsageErrorf("no column updates given")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("datasets", args[0], "columns")
			if err != nil {
				return err
			}

			var result json.RawMessage
			err = c.Patch(cmd.Context(), path, map[string]interface{}{"column_updates": columnUpdates}, &result)
			if err != nil {
				return err
			}
			return f.PrintMutation(result, "Columns updated.", columnUpdateColumns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&key, "key", "", "Column key to update")
	flags.StringVar(&label, "label", "", "Human-readable label (empty clears it)")
	flags.StringVar(&description, "description", "", "Column description (empty clears it)")
	flags.StringVar(&updates, "updates", "", "JSON array of {key,label,description}, @file or - for stdin")
	return cmd
}
