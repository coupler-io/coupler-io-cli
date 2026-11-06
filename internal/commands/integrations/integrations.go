// Package integrations lists the source and destination integrations
// (connectors) available in Coupler.io and their configurable params.
package integrations

import (
	"encoding/json"
	"fmt"
	"net/url"
	"strings"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var listColumns = []output.Column{
	{Header: "KEY", Field: "key"},
	{Header: "NAME", Field: "name"},
}

var detailColumns = []output.Column{
	{Header: "KEY", Field: "key"},
	{Header: "NAME", Field: "name"},
	{Header: "CREDENTIAL_PROVIDER", Field: "credential_provider"},
}

var paramColumns = []output.Column{
	{Header: "NAME", Field: "name"},
	{Header: "TITLE", Field: "title"},
	{Header: "TYPE", Field: "type"},
	{Header: "REQUIRED", Field: "required"},
	{Header: "HELP", Field: "helper_text"},
}

func NewCmdIntegrations(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "integrations",
		Aliases: []string{"integration"},
		Short:   "Browse available source and destination integrations",
	}
	cmd.AddCommand(newCmdKind(f, "source", true))
	cmd.AddCommand(newCmdKind(f, "destination", false))
	return cmd
}

// newCmdKind builds `integrations sources` or `integrations destinations`.
func newCmdKind(f *cmdutil.Factory, kind string, withFieldOptions bool) *cobra.Command {
	cmd := &cobra.Command{
		Use:     kind + "s",
		Aliases: []string{kind},
		Short:   fmt.Sprintf("Browse %s integrations", kind),
	}
	cmd.AddCommand(newCmdList(f, kind))
	cmd.AddCommand(newCmdGet(f, kind))
	if withFieldOptions {
		cmd.AddCommand(newCmdFieldOptions(f))
	}
	return cmd
}

func newCmdList(f *cmdutil.Factory, kind string) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: fmt.Sprintf("List %s integrations", kind),
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), cmdutil.Path("integrations", kind+"s"), nil, &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{Items: "integrations"}, listColumns)
		},
	}
}

func newCmdGet(f *cmdutil.Factory, kind string) *cobra.Command {
	return &cobra.Command{
		Use:   "get <key>",
		Short: fmt.Sprintf("Show a %s integration and the params it accepts", kind),
		Long: fmt.Sprintf("Shows the params you can set when creating a %[1]s with this integration. "+
			"Use --output json to also see llm_instructions and option lists.", kind),
		Example: fmt.Sprintf("  coupler integrations %ss get csv", kind),
		Args:    cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), cmdutil.Path("integrations", kind+"s", args[0]), nil, &body); err != nil {
				return err
			}
			return f.PrintDetail(body, detailColumns,
				cmdutil.Section{Title: "Params", Key: "params", Columns: paramColumns})
		},
	}
}

func newCmdFieldOptions(f *cmdutil.Factory) *cobra.Command {
	var field string
	var credentialID int
	var params []string

	cmd := &cobra.Command{
		Use:   "field-options <key>",
		Short: "List the allowed values of a source param (e.g. accounts, sheets)",
		Long: "Some params only accept values that depend on the connected account, such as a list of " +
			"ad accounts. This resolves them live from the source. Use `integrations sources get` to see " +
			"which params have `options_requires_credential` or `options_depends_on`.",
		Example: `  coupler integrations sources field-options gsheet --field sheetsList --credential-id 15 \
      --param spreadsheetURL=https://docs.google.com/spreadsheets/d/...`,
		Args: cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := cmdutil.RequireFlag("field", field); err != nil {
				return err
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("integrations", "sources", args[0], "field_options")
			if err != nil {
				return err
			}

			query := url.Values{"field": {field}}
			if credentialID != 0 {
				query.Set("credential_id", fmt.Sprint(credentialID))
			}
			for _, pair := range params {
				key, value, ok := strings.Cut(pair, "=")
				if !ok || key == "" {
					return cmdutil.UsageErrorf("--param %q must be in key=value form", pair)
				}
				query.Set("params["+key+"]", value)
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, query, &body); err != nil {
				return err
			}
			// Option objects differ per field, so print them as JSON.
			return f.PrintJSON(body)
		},
	}

	cmd.Flags().StringVar(&field, "field", "", "Param name to resolve options for (required)")
	cmd.Flags().IntVar(&credentialID, "credential-id", 0, "Credential used to load the options")
	cmd.Flags().StringArrayVar(&params, "param", nil, "Value of another param the options depend on, as key=value (repeatable)")
	return cmd
}
