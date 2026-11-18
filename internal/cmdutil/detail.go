package cmdutil

import (
	"encoding/json"
	"fmt"

	"github.com/coupler-io/coupler-io-cli/internal/output"
)

// Section is a nested array of an object, printed as its own table.
type Section struct {
	Title   string
	Key     string // key of the array in the object
	Columns []output.Column
}

// PrintDetail prints one object followed by its nested sections.
//
//   - json: the untouched body
//   - csv: the first section only (the object itself when there are none)
//   - table/text: the object, then each section under its title
func (f *Factory) PrintDetail(body json.RawMessage, columns []output.Column, sections ...Section) error {
	format := f.Format()
	out := f.IOStreams.Out

	if format == output.FormatJSON {
		return output.Print(out, format, body, nil)
	}

	var envelope map[string]json.RawMessage
	if len(sections) > 0 {
		if err := json.Unmarshal(body, &envelope); err != nil {
			return fmt.Errorf("data must be a JSON object")
		}
	}

	if format == output.FormatCSV {
		if len(sections) == 0 {
			return output.Print(out, format, body, columns)
		}
		return output.Print(out, format, sectionItems(envelope[sections[0].Key]), sections[0].Columns)
	}

	if err := output.Print(out, format, body, columns); err != nil {
		return err
	}
	for _, s := range sections {
		items := sectionItems(envelope[s.Key])
		fmt.Fprintf(out, "\n%s:\n", s.Title)
		if string(items) == "[]" {
			fmt.Fprintln(out, "  (none)")
			continue
		}
		if err := output.Print(out, format, items, s.Columns); err != nil {
			return err
		}
	}
	return nil
}

// sectionItems turns a missing or null section into an empty array.
func sectionItems(raw json.RawMessage) json.RawMessage {
	if len(raw) == 0 || string(raw) == "null" {
		return json.RawMessage("[]")
	}
	var items []json.RawMessage
	if json.Unmarshal(raw, &items) == nil && len(items) == 0 {
		return json.RawMessage("[]")
	}
	return raw
}
