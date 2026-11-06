package output

import (
	"fmt"
	"io"
)

// TextFormatter prints "Header: value" lines, one block per item.
type TextFormatter struct{}

func (f *TextFormatter) Format(w io.Writer, data interface{}, columns []Column) error {
	items, _, err := toItems(data)
	if err != nil {
		return err
	}

	for i, item := range items {
		if i > 0 {
			fmt.Fprintln(w, "---")
		}
		for _, col := range columns {
			fmt.Fprintf(w, "%s: %s\n", col.Header, formatValue(lookupField(item, col.Field)))
		}
	}
	return nil
}
