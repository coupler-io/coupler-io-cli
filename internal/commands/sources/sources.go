package sources

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

func NewCmdSources(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "sources",
		Aliases: []string{"source"},
		Short:   "Add and configure dataflow sources",
		Long: "A source pulls data into a dataflow. Find integration keys with `coupler integrations sources list`, " +
			"their params with `coupler integrations sources get <key>`, and credentials with `coupler credentials list`.",
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
		Short: "Add a source to a dataflow",
		Example: `  coupler sources create --dataflow-id <id> --integration csv --param url=https://example.com/data.csv
  coupler sources create --dataflow-id <id> --integration gsheet --credential-id 15 --params @gsheet.json`,
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
			path, err := f.WorkspacePath("dataflows", dataflowID, "sources")
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
			return f.PrintMutation(result, "Source created.", columns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&dataflowID, "dataflow-id", "", "Dataflow to add the source to (required)")
	flags.StringVar(&integration, "integration", "", "Source integration key, e.g. csv or gsheet (required)")
	flags.IntVar(&credentialID, "credential-id", 0, "Credential to connect (see `coupler credentials list`)")
	flags.StringArrayVar(&params, "param", nil, "Integration param as key=value (repeatable, always a string)")
	flags.StringVar(&paramsJSON, "params", "", "Integration params as a JSON object, @file or - for stdin")
	return cmd
}

func newCmdUpdate(f *cmdutil.Factory) *cobra.Command {
	var (
		name, paramsJSON, lifecycleState string
		credentialID                     int
		enabled                          bool
		params                           []string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: "Update a source",
		Long: "Only the fields you pass are changed. Params are merged into the current values. " +
			"Pausing needs a successful run and a plan that includes pausing.",
		Example: `  coupler sources update <id> --param url=https://example.com/new.csv
  coupler sources update <id> --lifecycle-state paused
  coupler sources update <id> --enabled=false`,
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
			if flags.Changed("enabled") {
				body["enabled"] = enabled
			}
			if flags.Changed("lifecycle-state") {
				if !contains(lifecycleStates, lifecycleState) {
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
			path, err := f.WorkspacePath("sources", args[0])
			if err != nil {
				return err
			}

			var result json.RawMessage
			if err := c.Patch(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, fmt.Sprintf("Source %s updated.", args[0]), columns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&name, "name", "", "Source name")
	flags.IntVar(&credentialID, "credential-id", 0, "Credential to connect")
	flags.BoolVar(&enabled, "enabled", true, "Whether the source runs")
	flags.StringVar(&lifecycleState, "lifecycle-state", "", "active, paused or disabled")
	flags.StringArrayVar(&params, "param", nil, "Integration param as key=value (repeatable, always a string)")
	flags.StringVar(&paramsJSON, "params", "", "Integration params as a JSON object, @file or - for stdin")
	return cmd
}

func contains(list []string, v string) bool {
	for _, s := range list {
		if s == v {
			return true
		}
	}
	return false
}
