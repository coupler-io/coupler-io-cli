package datasets

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var snapshotListColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "DATAFLOW", Field: "dataflow_name"},
	{Header: "ROWS", Field: "rows_count"},
	{Header: "LAST_SUCCESS_RUN_AT", Field: "last_success_run_at"},
	{Header: "TABLE", Field: "table_name"},
	{Header: "SNAPSHOT_ID", Field: "last_dataset_snapshot_id"},
}

var snapshotDetailColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "DATASET_ID", Field: "dataset_id"},
	{Header: "TYPE", Field: "type"},
	{Header: "AI_CONTEXT", Field: "ai_context"},
}

func newCmdSnapshots(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "snapshots",
		Aliases: []string{"snapshot"},
		Short:   "Browse datasets that have data (latest successful snapshots)",
		Long:    "A snapshot is the dataset as of its last successful run, with the schema of that run.",
	}
	cmd.AddCommand(newCmdSnapshotsList(f))
	cmd.AddCommand(newCmdSnapshotsGet(f))
	return cmd
}

func newCmdSnapshotsList(f *cmdutil.Factory) *cobra.Command {
	var fl filters

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List dataset snapshots",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataset_snapshots")
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, fl.values(), &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{Items: "datasets", Footers: []output.Footer{totalFooter}}, snapshotListColumns)
		},
	}

	fl.bind(cmd)
	return cmd
}

func newCmdSnapshotsGet(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "get <id>",
		Short: "Show a snapshot and its columns",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataset_snapshots", args[0])
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, nil, &body); err != nil {
				return err
			}
			return f.PrintDetail(body, snapshotDetailColumns,
				cmdutil.Section{Title: "Columns", Key: "columns", Columns: columnColumns})
		},
	}
}
