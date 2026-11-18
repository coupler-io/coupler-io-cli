// Package dataflowparts builds the command groups for the two kinds of
// dataflow building blocks, sources and destinations. They share the same
// create/update shape, so they are described by a Spec instead of being
// written twice.
package dataflowparts

import (
	"encoding/json"
	"fmt"
	"strings"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

// Spec describes one kind of dataflow part.
type Spec struct {
	Noun   string // singular, lower case: "source"
	Plural string // plural, also the API path segment: "sources"

	// Long is the help text of the group command.
	Long string
	// IntegrationExample is shown in the --integration help, e.g. "csv or gsheet".
	IntegrationExample string
	// CreateExample and UpdateExample are the help examples for the subcommands.
	CreateExample string
	UpdateExample string

	// SupportsEnabled adds `update --enabled`. Only set it when the API's
	// PATCH endpoint accepts `enabled` for this kind; the API ignores unknown
	// fields, so a flag it does not read would silently do nothing.
	SupportsEnabled bool
}

var columns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "INTEGRATION", Field: "integration.key"},
	{Header: "ENABLED", Field: "enabled"},
	{Header: "STATE", Field: "lifecycle_state"},
	{Header: "CONFIGURED", Field: "params_configured"},
	{Header: "MISSING", Field: "params_remaining_errors"},
}

// NewCmd returns the command group (`sources` or `destinations`) for spec.
func NewCmd(f *cmdutil.Factory, spec Spec) *cobra.Command {
	cmd := &cobra.Command{
		Use:     spec.Plural,
		Aliases: []string{spec.Noun},
		Short:   fmt.Sprintf("Add and configure dataflow %s", spec.Plural),
		Long:    spec.Long,
	}
	cmd.AddCommand(newCmdCreate(f, spec))
	cmd.AddCommand(newCmdUpdate(f, spec))
	return cmd
}

func newCmdCreate(f *cmdutil.Factory, spec Spec) *cobra.Command {
	var (
		dataflowID, integration, paramsJSON string
		credentialID                        int
		params                              []string
	)

	cmd := &cobra.Command{
		Use:     "create",
		Short:   fmt.Sprintf("Add a %s to a dataflow", spec.Noun),
		Example: spec.CreateExample,
		Args:    cobra.NoArgs,
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
			path, err := f.WorkspacePath("dataflows", dataflowID, spec.Plural)
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
			return f.PrintMutation(result, capitalize(spec.Noun)+" created.", columns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&dataflowID, "dataflow-id", "", fmt.Sprintf("Dataflow to add the %s to (required)", spec.Noun))
	flags.StringVar(&integration, "integration", "", fmt.Sprintf("%s integration key, e.g. %s (required)", capitalize(spec.Noun), spec.IntegrationExample))
	flags.IntVar(&credentialID, "credential-id", 0, "Credential to connect (see `coupler-io credentials list`)")
	addParamFlags(cmd, &params, &paramsJSON)
	return cmd
}

func newCmdUpdate(f *cmdutil.Factory, spec Spec) *cobra.Command {
	var (
		name, paramsJSON, lifecycleState string
		credentialID                     int
		enabled                          bool
		params                           []string
	)

	cmd := &cobra.Command{
		Use:   "update <id>",
		Short: fmt.Sprintf("Update a %s", spec.Noun),
		Long: "Only the fields you pass are changed. Params are merged into the current values. " +
			"Pausing needs a successful run and a plan that includes pausing.",
		Example: spec.UpdateExample,
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			flags := cmd.Flags()
			body := map[string]interface{}{}

			if flags.Changed("name") {
				body["name"] = name
			}
			if flags.Changed("credential-id") {
				body["credential_id"] = credentialID
			}
			if spec.SupportsEnabled && flags.Changed("enabled") {
				body["enabled"] = enabled
			}
			if flags.Changed("lifecycle-state") {
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
				return cmdutil.UsageErrorf("nothing to update: set at least one flag (see --help)")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath(spec.Plural, args[0])
			if err != nil {
				return err
			}

			var result json.RawMessage
			if err := c.Patch(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, fmt.Sprintf("%s %s updated.", capitalize(spec.Noun), args[0]), columns)
		},
	}

	flags := cmd.Flags()
	flags.StringVar(&name, "name", "", capitalize(spec.Noun)+" name")
	flags.IntVar(&credentialID, "credential-id", 0, "Credential to connect")
	if spec.SupportsEnabled {
		flags.BoolVar(&enabled, "enabled", true, fmt.Sprintf("Whether the %s runs", spec.Noun))
	}
	flags.StringVar(&lifecycleState, "lifecycle-state", "", "active, paused or disabled")
	addParamFlags(cmd, &params, &paramsJSON)
	return cmd
}

func addParamFlags(cmd *cobra.Command, params *[]string, paramsJSON *string) {
	cmd.Flags().StringArrayVar(params, "param", nil, "Integration param as key=value (repeatable, always a string)")
	cmd.Flags().StringVar(paramsJSON, "params", "", "Integration params as a JSON object, @file or - for stdin")
}

func capitalize(s string) string {
	if s == "" {
		return s
	}
	return strings.ToUpper(s[:1]) + s[1:]
}
