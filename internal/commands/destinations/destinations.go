package destinations

import (
	"encoding/json"
	"fmt"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var columns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "INTEGRATION", Field: "integration.key"},
	{Header: "ENABLED", Field: "enabled"},
	{Header: "STATE", Field: "lifecycle_state"},
	{Header: "CONFIGURED", Field: "params_configured"},
	{Header: "MISSING", Field: "params_remaining_errors"},
}

var lifecycleStates = []string{"active", "paused", "disabled"}

func NewCmdDestinations(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "destinations",
		Aliases: []string{"destination"},
		Short:   "Add and configure dataflow destinations",
		Long: "A destination receives a dataflow's data (Google Sheets, BigQuery, ...). Find integration keys with " +
			"`coupler integrations destinations list` and their params with `coupler integrations destinations get <key>`.",
	}
	cmd.AddCommand(newCmdCreate(f))
	cmd.AddCommand(newCmdUpdate(f))
	return cmd
}

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var (
		dataflowID, integration, paramsJSON string
		credentialID                        int
		params                              []string
	)

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Add a destination to a dataflow",
		Example: `  coupler destinations create --dataflow-id <id> --integration google_spreadsheet \
      --credential-id 15 --param sheet_name=Sales`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireFlag("dataflow-id", dataflowID); err != nil {
				return err
			}
			if err := cmdutil.RequireFlag("integration", integration); err != nil {
				return err
			}
			parsed, err := f.ParseParams(paramsJSON, params)
			if err != nil {
				return err
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows", dataflowID, "destinations")
			if err != nil {
				return err
			}

			body := map[string]interface{}{"integration_key": integration}
			if credentialID != 0 {
				body["credential_id"] = credentialID
			}
			if parsed != nil {
				body["params"] = parsed
			}

			var result json.RawMessage
			if err := c.Post(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, "Destination created.", columns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&dataflowID, "dataflow-id", "", "Dataflow to add the destination to (required)")
	flags.StringVar(&integration, "integration", "", "Destination integration key, e.g. google_spreadsheet (required)")
	flags.IntVar(&credentialID, "credential-id", 0, "Credential to connect (see `coupler credentials list`)")
	flags.StringArrayVar(&params, "param", nil, "Integration param as key=value (repeatable, always a string)")
	flags.StringVar(&paramsJSON, "params", "", "Integration params as a JSON object, @file or - for stdin")
	return cmd
}

func newCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var (
		name, paramsJSON, lifecycleState string
		credentialID                     int
		params                           []string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a destination",
		Long: "Only the fields you pass are changed. Params are merged into the current values. " +
			"Pausing needs a successful run and a plan that includes pausing.",
		Example: `  coupler destinations update <id> --param sheet_name=Q4
  coupler destinations update <id> --lifecycle-state paused`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			body := map[string]interface{}{}

			if flags.Changed("name") {
				body["name"] = name
			}
			if flags.Changed("credential-id") {
				body["credential_id"] = credentialID
			}
			if flags.Changed("lifecycle-state") {
				valid := false
				for _, s := range lifecycleStates {
					valid = valid || s == lifecycleState
				}
				if !valid {
					return cmdutil.UsageErrorf("--lifecycle-state must be one of: %v", lifecycleStates)
				}
				body["lifecycle_state"] = lifecycleState
			}
			parsed, err := f.ParseParams(paramsJSON, params)
			if err != nil {
				return err
			}
			if parsed != nil {
				body["params"] = parsed
			}
			if len(body) == 0 {
				return fmt.Errorf("nothing to update: set at least one flag (see --help)")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("destinations", args[0])
			if err != nil {
				return err
			}

			var result json.RawMessage
			if err := c.Patch(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, fmt.Sprintf("Destination %s updated.", args[0]), columns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&name, "name", "", "Destination name")
	flags.IntVar(&credentialID, "credential-id", 0, "Credential to connect")
	flags.StringVar(&lifecycleState, "lifecycle-state", "", "active, paused or disabled")
	flags.StringArrayVar(&params, "param", nil, "Integration param as key=value (repeatable, always a string)")
	flags.StringVar(&paramsJSON, "params", "", "Integration params as a JSON object, @file or - for stdin")
	return cmd
}
