// Package destinations defines the `coupler destinations` command group.
package destinations

import (
	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/commands/dataflowparts"
)

func NewCmdDestinations(f *cmdutil.Factory) *cobra.Command {
	return dataflowparts.NewCmd(f, dataflowparts.Spec{
		Noun:   "destination",
		Plural: "destinations",
		Long: "A destination receives a dataflow's data (Google Sheets, BigQuery, ...). Find integration keys with " +
			"`coupler integrations destinations list` and their params with `coupler integrations destinations get <key>`.",
		IntegrationExample: "google_spreadsheet",
		CreateExample: `  coupler destinations create --dataflow-id <id> --integration google_spreadsheet \
      --credential-id 15 --param sheet_name=Sales`,
		UpdateExample: `  coupler destinations update <id> --param sheet_name=Q4
  coupler destinations update <id> --lifecycle-state paused`,
		// No `update --enabled`: PATCH /v1/workspaces/{id}/destinations/{id} does
		// not accept `enabled` (only name, credential_id, params and
		// lifecycle_state). Use --lifecycle-state disabled to switch one off.
		SupportsEnabled: false,
	})
}
