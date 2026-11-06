package dataflows

import (
	"encoding/json"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newCmdCreate(f *cmdutil.Factory) *cobra.Command {
	var name, templateID string

	cmd := &cobra.Command{
		Use:   "create",
		Short: "Create a dataflow, empty or from a template",
		Example: `  coupler dataflows create --name "Sales report"
  coupler dataflows create --template-id <id>   # ids from: coupler templates list`,
		Args: cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			if name == "" && templateID == "" {
				return cmdutil.UsageErrorf("--name is required unless --template-id is set")
			}

			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("dataflows")
			if err != nil {
				return err
			}

			body := map[string]interface{}{}
			if name != "" {
				body["name"] = name
			}
			if templateID != "" {
				body["template_id"] = templateID
			}

			var result json.RawMessage
			if err := c.Post(cmd.Context(), path, body, &result); err != nil {
				return err
			}
			return f.PrintMutation(result, "Dataflow created.", createColumns)
		},
	}

	cmd.Flags().StringVar(&name, "name", "", "Dataflow name (required unless --template-id is set)")
	cmd.Flags().StringVar(&templateID, "template-id", "", "Copy this template (see `coupler templates list`)")
	return cmd
}
