package output_test

import (
	"bytes"
	"encoding/json"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/output"
)

var cols = []output.Column{
	{Header: "ID", Field: "id"},
	{Header: "NAME", Field: "name"},
	{Header: "KEY", Field: "integration.key"},
}

const listJSON = `[{"id":10000000000000001,"name":"Sales","integration":{"key":"csv"}},{"id":2,"name":null}]`

func render(t *testing.T, f output.Format, data string) string {
	t.Helper()
	var buf bytes.Buffer
	if err := output.Print(&buf, f, json.RawMessage(data), cols); err != nil {
		t.Fatal(err)
	}
	return buf.String()
}

func TestTableList(t *testing.T) {
	out := render(t, output.FormatTable, listJSON)
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 3 {
		t.Fatalf("expected header + 2 rows, got:\n%s", out)
	}
	if !strings.HasPrefix(lines[0], "ID") || !strings.Contains(lines[1], "10000000000000001") || !strings.Contains(lines[1], "csv") {
		t.Errorf("unexpected table:\n%s", out)
	}
}

func TestTableSingleObjectIsVertical(t *testing.T) {
	out := render(t, output.FormatTable, `{"id":7,"name":"One","integration":{"key":"gsheet"}}`)
	for _, want := range []string{"ID", "7", "NAME", "One", "KEY", "gsheet"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Count(out, "\n") != 3 {
		t.Errorf("expected one line per column, got:\n%s", out)
	}
}

func TestTableTruncatesLongCells(t *testing.T) {
	long := strings.Repeat("x", 200)
	out := render(t, output.FormatTable, `[{"id":1,"name":"`+long+`"}]`)
	if strings.Contains(out, long) || !strings.Contains(out, "...") {
		t.Errorf("long value not truncated:\n%s", out)
	}
}

func TestEmptyCollectionsRenderBlank(t *testing.T) {
	out := render(t, output.FormatCSV, `[{"id":1,"name":[],"integration":{}}]`)
	if !strings.Contains(out, "\n1,,\n") {
		t.Errorf("unexpected csv: %q", out)
	}
}

func TestCSV(t *testing.T) {
	out := render(t, output.FormatCSV, `[{"id":1,"name":"a,b \"q\""}]`)
	want := "ID,NAME,KEY\n1,\"a,b \"\"q\"\"\",\n"
	if out != want {
		t.Errorf("csv = %q, want %q", out, want)
	}
}

func TestText(t *testing.T) {
	out := render(t, output.FormatText, listJSON)
	if !strings.Contains(out, "ID: 10000000000000001\nNAME: Sales\nKEY: csv\n---\nID: 2") {
		t.Errorf("unexpected text:\n%s", out)
	}
}

func TestJSONIsIndented(t *testing.T) {
	out := render(t, output.FormatJSON, `{"a":1}`)
	if out != "{\n  \"a\": 1\n}\n" {
		t.Errorf("json = %q", out)
	}
}

func TestParseFormat(t *testing.T) {
	if f, err := output.ParseFormat("csv"); err != nil || f != output.FormatCSV {
		t.Errorf("csv: %v %v", f, err)
	}
	if _, err := output.ParseFormat("yaml"); err == nil || !strings.Contains(err.Error(), "table, json, text, csv") {
		t.Errorf("expected helpful error, got %v", err)
	}
}

func TestPrintListEnvelopeAndFooters(t *testing.T) {
	body := json.RawMessage(`{"meta":{"total_count":2},"things":[{"id":1,"name":"a"}],"setup_url":"https://x"}`)
	list := output.List{Items: "things", Footers: []output.Footer{
		{Label: "Total", Path: []string{"meta", "total_count"}},
		{Label: "Setup", Path: []string{"setup_url"}},
		{Label: "Missing", Path: []string{"nope"}},
	}}

	var table bytes.Buffer
	if err := output.PrintList(&table, output.FormatTable, body, list, cols); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(table.String(), "\nTotal: 2\nSetup: https://x\n") || strings.Contains(table.String(), "Missing") {
		t.Errorf("unexpected table:\n%s", table.String())
	}

	var csvOut bytes.Buffer
	if err := output.PrintList(&csvOut, output.FormatCSV, body, list, cols); err != nil {
		t.Fatal(err)
	}
	if strings.Contains(csvOut.String(), "Total") {
		t.Errorf("csv must not contain footers:\n%s", csvOut.String())
	}

	var jsonOut bytes.Buffer
	if err := output.PrintList(&jsonOut, output.FormatJSON, body, list, cols); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(jsonOut.String(), `"meta"`) || !strings.Contains(jsonOut.String(), `"setup_url"`) {
		t.Errorf("json must keep the full body:\n%s", jsonOut.String())
	}
}

func TestPrintListTopLevelArray(t *testing.T) {
	var buf bytes.Buffer
	if err := output.PrintList(&buf, output.FormatTable, json.RawMessage(listJSON), output.List{}, cols); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(buf.String(), "Sales") {
		t.Errorf("unexpected output:\n%s", buf.String())
	}
}

func TestRowColumnsPreservesOrderAndUnion(t *testing.T) {
	columns, err := output.RowColumns(json.RawMessage(`[{"z":1,"a":2,"Row.0":3},{"a":1,"extra":2}]`))
	if err != nil {
		t.Fatal(err)
	}
	var got []string
	for _, c := range columns {
		got = append(got, c.Field)
	}
	if strings.Join(got, ",") != "z,a,Row.0,extra" {
		t.Errorf("columns = %v", got)
	}

	// A dotted key must resolve as a literal key, not a path.
	var buf bytes.Buffer
	if err := output.Print(&buf, output.FormatCSV, json.RawMessage(`[{"Row.0":"v"}]`), columns[2:3]); err != nil {
		t.Fatal(err)
	}
	if buf.String() != "Row.0\nv\n" {
		t.Errorf("csv = %q", buf.String())
	}
}
