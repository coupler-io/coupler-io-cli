package cmdutil

import (
	"bytes"
	"encoding/json"
	"fmt"

	"github.com/coupler-io/coupler-io-cli/internal/output"
)

// PrintObject prints a single-object response body.
func (f *Factory) PrintObject(body json.RawMessage, columns []output.Column) error {
	return output.Print(f.IOStreams.Out, f.Format(), body, columns)
}

// PrintList prints a list response body (see output.PrintList).
func (f *Factory) PrintList(body json.RawMessage, list output.List, columns []output.Column) error {
	return output.PrintList(f.IOStreams.Out, f.Format(), body, list, columns)
}

// PrintMutation prints the result of a create/update/run call. When the API
// returned an object it is printed with columns; when it returned nothing (or
// something that is not an object) the human message is printed instead. With
// --output json the body, or {"message": ...}, is always valid JSON.
func (f *Factory) PrintMutation(body json.RawMessage, message string, columns []output.Column) error {
	hasObject := bytes.HasPrefix(bytes.TrimSpace(body), []byte("{"))

	switch {
	case hasObject && (columns != nil || f.Format() == output.FormatJSON):
		return f.PrintObject(body, columns)
	case f.Format() == output.FormatJSON:
		return output.Print(f.IOStreams.Out, output.FormatJSON, map[string]string{"message": message}, nil)
	default:
		_, err := fmt.Fprintln(f.IOStreams.Out, message)
		return err
	}
}

// PrintJSON prints a free-form response as indented JSON regardless of
// --output, for payloads with no stable column layout.
func (f *Factory) PrintJSON(body json.RawMessage) error {
	return output.Print(f.IOStreams.Out, output.FormatJSON, body, nil)
}
