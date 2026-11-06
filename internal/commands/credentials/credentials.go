package credentials

import (
	"encoding/json"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var columns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "TYPE", Field: "type"},
	{Header: "PROVIDER", Field: "provider"},
	{Header: "NAME", Field: "name"},
	{Header: "STATUS", Field: "status"},
}

func NewCmdCredentials(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "credentials",
		Aliases: []string{"credential"},
		Short:   "List connected credentials",
		Long: "Credentials (connected accounts) are created in the Coupler.io web app. " +
			"Use a credential id with `sources create --credential-id` or `destinations create --credential-id`.",
	}
	cmd.AddCommand(newCmdList(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var provider, credType string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List credentials in the workspace",
		Example: `  coupler credentials list --type source
  coupler credentials list --provider google`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if credType != "" && credType != "source" && credType != "destination" {
				return cmdutil.UsageErrorf("--type must be source or destination")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("credentials")
			if err != nil {
				return err
			}

			query := url.Values{}
			if provider != "" {
				query.Set("provider", provider)
			}
			if credType != "" {
				query.Set("type", credType)
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, query, &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{
				Items:   "credentials",
				Footers: []output.Footer{{Label: "Add credentials at", Path: []string{"setup_url"}}},
			}, columns)
		},
	}

	cmd.Flags().StringVar(&provider, "provider", "", "Filter by provider (e.g. google)")
	cmd.Flags().StringVar(&credType, "type", "", "Filter by type: source or destination")
	return cmd
}
