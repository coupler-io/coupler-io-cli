// Package output renders API responses as table, text, JSON or CSV.
package output

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"strings"
)

type Format string

const (
	FormatJSON  Format = "json"
	FormatTable Format = "table"
	FormatText  Format = "text"
	FormatCSV   Format = "csv"
)

// Formats lists the accepted values of --output.
var Formats = []Format{FormatTable, FormatJSON, FormatText, FormatCSV}

// ParseFormat validates a user supplied format name.
func ParseFormat(s string) (Format, error) {
	for _, f := range Formats {
		if string(f) == s {
			return f, nil
		}
	}
	names := make([]string, len(Formats))
	for i, f := range Formats {
		names[i] = string(f)
	}
	return "", fmt.Errorf("unknown output format %q (use one of: %s)", s, strings.Join(names, ", "))
}

// Column maps a response field to a column. Field is looked up as a literal
// key first, then as a dotted path (e.g. "integration.key").
type Column struct {
	Header string
	Field  string
}

type Formatter interface {
	Format(w io.Writer, data interface{}, columns []Column) error
}

func New(format Format) Formatter {
	switch format {
	case FormatJSON:
		return &JSONFormatter{}
	case FormatText:
		return &TextFormatter{}
	case FormatCSV:
		return &CSVFormatter{}
	default:
		return &TableFormatter{}
	}
}

func Print(w io.Writer, format Format, data interface{}, columns []Column) error {
	return New(format).Format(w, data, columns)
}

// toItems normalises data (struct, map, slice or json.RawMessage) into a list
// of objects. single reports whether data was one object rather than a list.
func toItems(data interface{}) (items []map[string]interface{}, single bool, err error) {
	raw, err := json.Marshal(data)
	if err != nil {
		return nil, false, err
	}

	dec := json.NewDecoder(bytes.NewReader(raw))
	dec.UseNumber() // keep large ids and exact decimals intact

	var v interface{}
	if err := dec.Decode(&v); err != nil {
		return nil, false, fmt.Errorf("data must be a JSON object or array")
	}

	switch t := v.(type) {
	case nil:
		return nil, false, nil
	case map[string]interface{}:
		return []map[string]interface{}{t}, true, nil
	case []interface{}:
		items = make([]map[string]interface{}, 0, len(t))
		for _, el := range t {
			m, ok := el.(map[string]interface{})
			if !ok {
				return nil, false, fmt.Errorf("data must be a JSON object or array of objects")
			}
			items = append(items, m)
		}
		return items, false, nil
	default:
		return nil, false, fmt.Errorf("data must be a JSON object or array")
	}
}

// lookupField returns the value for a column field in item.
func lookupField(item map[string]interface{}, field string) interface{} {
	if v, ok := item[field]; ok {
		return v
	}
	if !strings.Contains(field, ".") {
		return nil
	}
	var cur interface{} = item
	for _, key := range strings.Split(field, ".") {
		m, ok := cur.(map[string]interface{})
		if !ok {
			return nil
		}
		cur = m[key]
	}
	return cur
}

func formatValue(v interface{}) string {
	switch val := v.(type) {
	case nil:
		return ""
	case string:
		return val
	case json.Number:
		return val.String()
	case bool:
		if val {
			return "true"
		}
		return "false"
	case float64:
		if val == float64(int64(val)) {
			return fmt.Sprintf("%d", int64(val))
		}
		return fmt.Sprintf("%g", val)
	case map[string]interface{}:
		if len(val) == 0 {
			return ""
		}
	case []interface{}:
		if len(val) == 0 {
			return ""
		}
	}

	b, err := json.Marshal(v)
	if err != nil {
		return fmt.Sprintf("%v", v)
	}
	return string(b)
}
