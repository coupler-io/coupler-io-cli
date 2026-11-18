// Package sources defines the `coupler sources` command group.
package sources

import (
	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/commands/dataflowparts"
)

func NewCmdSources(f *cmdutil.Factory) *cobra.Command {
	return dataflowparts.NewCmd(f, dataflowparts.Spec{
		Noun:   "source",
		Plural: "sources",
		Long: "A source pulls data into a dataflow. Find integration keys with `coupler integrations sources list`, " +
			"their params with `coupler integrations sources get <key>`, and credentials with `coupler credentials list`.",
		IntegrationExample: "csv or gsheet",
		CreateExample: `  coupler sources create --dataflow-id <id> --integration csv --param url=https://example.com/data.csv
  coupler sources create --dataflow-id <id> --integration gsheet --credential-id 15 --params @gsheet.json`,
		UpdateExample: `  coupler sources update <id> --param url=https://example.com/new.csv
  coupler sources update <id> --lifecycle-state paused
  coupler sources update <id> --enabled=false`,
		SupportsEnabled: true,
	})
}
