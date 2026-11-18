package datasets

import (
	"net/url"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var listColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "DATAFLOW", Field: "dataflow_name"},
	{Header: "STATE", Field: "state"},
	{Header: "QUERYABLE", Field: "queryable"},
	{Header: "ROWS", Field: "rows_count"},
	{Header: "LAST_SUCCESS_RUN_AT", Field: "last_success_run_at"},
	{Header: "TABLE", Field: "table_name"},
}

var detailColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "DATAFLOW", Field: "dataflow_name"},
	{Header: "DATAFLOW_ID", Field: "dataflow_id"},
	{Header: "TYPE", Field: "type"},
	{Header: "STATE", Field: "state"},
	{Header: "QUERYABLE", Field: "queryable"},
	{Header: "ROWS", Field: "rows_count"},
	{Header: "LAST_SUCCESS_RUN_AT", Field: "last_success_run_at"},
	{Header: "LAST_ERROR", Field: "last_error"},
	{Header: "TABLE", Field: "table_name"},
	{Header: "QUERY", Field: "query"},
	{Header: "AI_CONTEXT", Field: "ai_context"},
	{Header: "PREVIEW_URL", Field: "preview_url"},
}

var columnColumns = []output.Column{
	{Header: "KEY", Field: "key"},
	{Header: "LABEL", Field: "label"},
	{Header: "DB_NAME", Field: "dbName"},
	{Header: "TYPE", Field: "schema.type"},
}

var totalFooter = output.Footer{Label: "Total", Path: []string{"meta", "total_count"}}

func NewCmdDatasets(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "datasets",
		Aliases: []string{"dataset"},
		Short:   "Browse, query and edit datasets",
		Long: "A dataset is a table produced by a dataflow. You can read its schema, run DuckDB SQL against " +
			"its latest data with `datasets query`, and add custom SQL datasets with `datasets create`.",
	}

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdGet(f))
	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdUpdate(f))
	cmd.AddCommand(newCmdQuery(f))
	cmd.AddCommand(newCmdColumns(f))
	cmd.AddCommand(newCmdSnapshots(f))

	return cmd
}

// filters are the search flags shared by `datasets list` and `datasets snapshots list`.
type filters struct {
	query, source, name, dataflowID string
}

func (fl *filters) bind(cmd *cobra.Command) {
	flags := cmd.Flags()
	flags.StringVar(&fl.query, "query", "", "Match dataset, dataflow and source names")
	flags.StringVar(&fl.source, "source", "", "Source integration key or name, e.g. gsheet")
	flags.StringVar(&fl.name, "name", "", "Match dataset and source names")
	flags.StringVar(&fl.dataflowID, "dataflow-id", "", "Only datasets of this dataflow")
}

func (fl *filters) values() url.Values {
	v := url.Values{}
	set := func(k, val string) {
		if val != "" {
			v.Set(k, val)
		}
	}
	set("query", fl.query)
	set("source", fl.source)
	set("name", fl.name)
	set("dataflow_id", fl.dataflowID)
	return v
}
