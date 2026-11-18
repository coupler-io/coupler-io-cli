package templates

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var columns = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "TYPE", Field: "type"},
	{Header: "SOURCES", Field: "sources"},
	{Header: "CATEGORIES", Field: "categories"},
}

func NewCmdTemplates(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "templates",
		Aliases: []string{"template"},
		Short:   "Browse dataflow templates",
	}
	cmd.AddCommand(newCmdList(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "list",
		Short: "List templates",
		Long:  "Lists available templates. Pass an id to `coupler-io dataflows create --template-id` to copy one.",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), cmdutil.Path("templates"), nil, &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{}, columns)
		},
	}
}
