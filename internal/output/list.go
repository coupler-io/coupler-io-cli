package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
)

// Footer is an extra "Label: value" line printed after a table or text list.
type Footer struct {
	Label string
	Path  []string // path of keys to the scalar in the response body
}

// List describes where a list response keeps its items. Items is the key of
// the array in the response envelope; empty means the body is the array.
type List struct {
	Items   string
	Footers []Footer
}

// PrintList prints a list response. JSON output is the untouched body, so
// scripts keep every field, totals and metadata. Table and text print the
// items followed by footers (e.g. totals); CSV prints items only.
func PrintList(w io.Writer, format Format, body json.RawMessage, list List, columns []Column) error {
	if format == FormatJSON {
		return Print(w, format, body, columns)
	}

	items := body
	if list.Items != "" {
		var envelope map[string]json.RawMessage
		if err := json.Unmarshal(body, &envelope); err != nil {
			return fmt.Errorf("data must be a JSON object")
		}
		items = envelope[list.Items]
	}
	if err := Print(w, format, items, columns); err != nil {
		return err
	}
	if format == FormatCSV {
		return nil
	}

	var lines []string
	for _, footer := range list.Footers {
		if v := lookupPath(body, footer.Path...); v != "" {
			lines = append(lines, footer.Label+": "+v)
		}
	}
	if len(lines) > 0 {
		fmt.Fprintln(w)
		for _, line := range lines {
			fmt.Fprintln(w, line)
		}
	}
	return nil
}

// lookupPath returns the scalar at path in body, or "" when it is missing.
func lookupPath(body json.RawMessage, path ...string) string {
	dec := json.NewDecoder(bytes.NewReader(body))
	dec.UseNumber()
	var v interface{}
	if dec.Decode(&v) != nil {
		return ""
	}
	for _, key := range path {
		m, ok := v.(map[string]interface{})
		if !ok {
			return ""
		}
		v = m[key]
	}
	return formatValue(v)
}

// RowColumns builds columns from the keys of an array of objects, in order of
// first appearance. Used for free-form results such as dataset queries.
func RowColumns(rows json.RawMessage) ([]Column, error) {
	var objects []json.RawMessage
	if err := json.Unmarshal(rows, &objects); err != nil {
		return nil, fmt.Errorf("rows must be a JSON array")
	}

	seen := map[string]bool{}
	var columns []Column
	for _, obj := range objects {
		dec := json.NewDecoder(bytes.NewReader(obj))
		if tok, err := dec.Token(); err != nil || tok != json.Delim('{') {
			return nil, fmt.Errorf("each row must be a JSON object")
		}
		for dec.More() {
			tok, err := dec.Token()
			if err != nil {
				return nil, err
			}
			key, _ := tok.(string)
			if !seen[key] {
				seen[key] = true
				columns = append(columns, Column{Header: key, Field: key})
			}
			var skip json.RawMessage
			if err := dec.Decode(&skip); err != nil {
				return nil, err
			}
		}
	}
	return columns, nil
}
