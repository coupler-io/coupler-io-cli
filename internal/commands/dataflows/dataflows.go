package dataflows

import (
	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var listColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "LAST_SUCCESS_RUN_AT", Field: "last_success_run_at"},
	{Header: "LAST_SUCCESSFUL_EXECUTION_ID", Field: "last_successful_execution_id"},
}

var detailColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "SCHEDULE", Field: "schedule"},
	{Header: "LAST_SUCCESSFUL_EXECUTION_ID", Field: "last_successful_execution_id"},
}

var sourceColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "ENABLED", Field: "enabled"},
	{Header: "STATE", Field: "lifecycle_state"},
	{Header: "CONFIGURED", Field: "params_configured"},
	{Header: "LAST_SUCCESS_RUN_AT", Field: "last_success_run_at"},
	{Header: "ERROR", Field: "error_details"},
}

var createColumns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "URL", Field: "url"},
}

func NewCmdDataflows(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "dataflows",
		Aliases: []string{"dataflow"},
		Short:   "Manage dataflows",
		Long: "A dataflow moves data from one or more sources to destinations on a schedule. " +
			"Add sources and destinations with `coupler sources` and `coupler destinations`.",
	}

	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdGet(f))
	cmd.AddCommand(newCmdSetup(f))
	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdUpdate(f))
	cmd.AddCommand(newCmdRun(f))

	return cmd
}
