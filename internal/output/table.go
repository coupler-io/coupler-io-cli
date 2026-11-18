package output

import (
	"fmt"
	"io"
	"strings"
	"text/tabwriter"
)

// maxCellWidth keeps wide values (nested JSON, long SQL) from wrecking the
// table layout. Use --output json or csv for full values.
const maxCellWidth = 80

type TableFormatter struct{}

func (f *TableFormatter) Format(w io.Writer, data interface{}, columns []Column) error {
	items, single, err := toItems(data)
	if err != nil {
		return err
	}

	tw := tabwriter.NewWriter(w, 0, 0, 2, ' ', 0)

	if single && len(items) == 1 {
		// One object: show it vertically as FIELD / VALUE pairs.
		for _, col := range columns {
			fmt.Fprintf(tw, "%s\t%s\n", col.Header, cell(lookupField(items[0], col.Field)))
		}
		return tw.Flush()
	}

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	fmt.Fprintln(tw, strings.Join(headers, "\t"))

	for _, item := range items {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = cell(lookupField(item, col.Field))
		}
		fmt.Fprintln(tw, strings.Join(row, "\t"))
	}
	return tw.Flush()
}

func cell(v interface{}) string {
	s := formatValue(v)
	s = strings.NewReplacer("\r\n", " ", "\n", " ", "\t", " ").Replace(s)
	if r := []rune(s); len(r) > maxCellWidth {
		s = string(r[:maxCellWidth-3]) + "..."
	}
	return s
}
