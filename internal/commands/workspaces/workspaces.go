package workspaces

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var columns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "ORGANIZATION", Field: "organization"},
}

func NewCmdWorkspaces(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "workspaces",
		Aliases: []string{"workspace"},
		Short:   "List the workspaces your token can access",
	}
	cmd.AddCommand(newCmdList(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List workspaces",
		Long:  "Lists workspaces you can access. Use an id as --workspace-id (or run `coupler-io configure`).",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), cmdutil.Path("workspaces"), nil, &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{Items: "workspaces"}, columns)
		},
	}
}
