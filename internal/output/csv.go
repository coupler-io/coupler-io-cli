package output

import (
	"encoding/csv"
	"io"
)

// CSVFormatter writes a header row followed by one row per item.
type CSVFormatter struct{}

func (f *CSVFormatter) Format(w io.Writer, data interface{}, columns []Column) error {
	items, _, err := toItems(data)
	if err != nil {
		return err
	}

	cw := csv.NewWriter(w)

	headers := make([]string, len(columns))
	for i, col := range columns {
		headers[i] = col.Header
	}
	if err := cw.Write(headers); err != nil {
		return err
	}

	for _, item := range items {
		row := make([]string, len(columns))
		for i, col := range columns {
			row[i] = formatValue(lookupField(item, col.Field))
		}
		if err := cw.Write(row); err != nil {
			return err
		}
	}

	cw.Flush()
	return cw.Error()
}
