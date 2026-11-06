package skills

import (
	"encoding/json"
	"fmt"
	"net/url"

	"github.com/spf13/cobra"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var columns = []output.Column{
	{Header: "SLUG", Field: "slug"},
	{Header: "NAME", Field: "name"},
}

func NewCmdSkills(f *cmdutil.Factory) *cobra.Command {
	cmd := &cobra.Command{
		Use:     "skills",
		Aliases: []string{"skill"},
		Short:   "List and read agent skills",
		Long:    "Agent skills are ready-made workflows for AI agents working with Coupler.io data.",
	}
	cmd.AddCommand(newCmdList(f))
	cmd.AddCommand(newCmdPrompt(f))
	return cmd
}

func newCmdList(f *cmdutil.Factory) *cobra.Command {
	var query string

	cmd := &cobra.Command{
		Use:   "list",
		Short: "List agent skills",
		Args:  cobra.NoArgs,
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("agent_skills")
			if err != nil {
				return err
			}

			params := url.Values{}
			if query != "" {
				params.Set("query", query)
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, params, &body); err != nil {
				return err
			}
			return f.PrintList(body, output.List{Items: "skills"}, columns)
		},
	}

	cmd.Flags().StringVar(&query, "query", "", "Search skill slugs, descriptions and categories")
	return cmd
}

func newCmdPrompt(f *cmdutil.Factory) *cobra.Command {
	return &cobra.Command{
		Use:   "prompt <slug>",
		Short: "Print the full instructions of a skill",
		Args:  cobra.ExactArgs(1),
		RunE: func(cmd *cobra.Command, args []string) error {
			c, err := f.NewClient()
			if err != nil {
				return err
			}
			path, err := f.WorkspacePath("agent_skills", args[0], "prompt")
			if err != nil {
				return err
			}

			var body json.RawMessage
			if err := c.Get(cmd.Context(), path, nil, &body); err != nil {
				return err
			}

			if f.Format() == output.FormatJSON {
				return f.PrintJSON(body)
			}
			var resp struct {
				Content string `json:"content"`
			}
			if err := json.Unmarshal(body, &resp); err != nil {
				return fmt.Errorf("unmarshal response: %w", err)
			}
			_, err = fmt.Fprintln(f.IOStreams.Out, resp.Content)
			return err
		},
	}
}
