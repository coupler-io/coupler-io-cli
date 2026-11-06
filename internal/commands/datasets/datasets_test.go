package datasets_test

import (
	"net/http"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/commands/datasets"
)

func TestListFiltersAndTotal(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/v1/workspaces/42/datasets" {
			t.Errorf("unexpected path %s", r.URL.Path)
		}
		q := r.URL.Query()
		if q.Get("source") != "gsheet" || q.Get("dataflow_id") != "df-1" || q.Has("name") {
			t.Errorf("unexpected query %v", q)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{
			"meta": map[string]int{"total_count": 1},
			"datasets": []map[string]interface{}{
				{"id": "ds-1", "name": "Sales", "dataflow_name": "Sales flow", "state": "FULL", "queryable": true, "rows_count": 120, "table_name": "abc12"},
			},
			"legend": map[string]interface{}{"state": map[string]string{}},
		})
	})

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "list", "--source", "gsheet", "--dataflow-id", "df-1"); err != nil {
		t.Fatal(err)
	}
	out := env.Out.String()
	for _, want := range []string{"ds-1", "Sales flow", "FULL", "120", "abc12", "Total: 1"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if strings.Contains(out, "legend") {
		t.Errorf("legend should only appear in JSON output:\n%s", out)
	}
}

func TestGetShowsColumns(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{
			"id": "ds-1", "name": "Sales", "state": "FULL", "queryable": true, "rows_count": 3, "ai_context": "# Sales",
			"columns": []map[string]interface{}{
				{"key": "Revenue", "label": "Revenue (USD)", "dbName": "col_0", "schema": map[string]string{"type": "number"}},
			},
		})
	})

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "get", "ds-1"); err != nil {
		t.Fatal(err)
	}
	out := env.Out.String()
	for _, want := range []string{"Sales", "# Sales", "Columns:", "Revenue (USD)", "col_0", "number"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
}

func TestQueryTable(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/workspaces/42/datasets/ds-1/query" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		if body := cmdtest.DecodeBody(t, r); body["statement"] != "select country, sum(x) as total from data group by 1" {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{
			"rows":      []map[string]interface{}{{"country": "DE", "total": 12.5}, {"country": "FR", "total": 3}},
			"truncated": false,
		})
	})

	err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "query", "ds-1",
		"-s", "select country, sum(x) as total from data group by 1")
	if err != nil {
		t.Fatal(err)
	}
	out := env.Out.String()
	for _, want := range []string{"country", "total", "DE", "12.5", "FR"} {
		if !strings.Contains(out, want) {
			t.Errorf("missing %q in:\n%s", want, out)
		}
	}
	if env.ErrOut.Len() != 0 {
		t.Errorf("unexpected stderr: %s", env.ErrOut.String())
	}
}

func TestQueryCSVAndTruncationWarning(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{
			"rows":      []map[string]interface{}{{"a": 1, "b": "x,y"}},
			"truncated": true,
		})
	})
	env.Factory.Viper.Set("output", "csv")

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "query", "ds-1", "-s", "select * from data"); err != nil {
		t.Fatal(err)
	}
	if env.Out.String() != "a,b\n1,\"x,y\"\n" {
		t.Errorf("csv output = %q", env.Out.String())
	}
	if !strings.Contains(env.ErrOut.String(), "truncated to 10,000 rows") {
		t.Errorf("expected truncation warning on stderr, got %q", env.ErrOut.String())
	}
}

func TestQueryNoRowsAndJSON(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 200, map[string]interface{}{"rows": []interface{}{}, "truncated": false})
	})

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "query", "ds-1", "-s", "select 1 where false"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "(no rows)") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}

	env.Out.Reset()
	env.Factory.Viper.Set("output", "json")
	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "query", "ds-1", "-s", "select 1 where false"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), `"truncated": false`) {
		t.Errorf("json output should be the raw response:\n%s", env.Out.String())
	}
}

func TestQueryStatementFromStdin(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if body := cmdtest.DecodeBody(t, r); body["statement"] != "select 1" {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"rows": []interface{}{}, "truncated": false})
	})
	env.Factory.IOStreams.In = strings.NewReader("select 1\n")

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "query", "ds-1", "-s", "-"); err != nil {
		t.Fatal(err)
	}
}

func TestQueryRequiresStatement(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) { t.Error("no request expected") })

	err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "query", "ds-1")
	if err == nil || !strings.Contains(err.Error(), "--statement is required") {
		t.Fatalf("got %v", err)
	}
}

func TestQueryTimeoutSurfaces(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		cmdtest.JSON(w, 504, map[string]interface{}{"error": map[string]string{"message": "Query timed out"}})
	})

	err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "query", "ds-1", "-s", "select 1")
	if err == nil || !strings.Contains(err.Error(), "504") || !strings.Contains(err.Error(), "Query timed out") {
		t.Fatalf("got %v", err)
	}
}

func TestSnapshots(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/v1/workspaces/42/dataset_snapshots":
			cmdtest.JSON(w, 200, map[string]interface{}{
				"meta":     map[string]int{"total_count": 1},
				"datasets": []map[string]interface{}{{"id": "ds-1", "name": "Sales", "dataflow_name": "Flow", "rows_count": 5, "last_dataset_snapshot_id": "snap-1"}},
			})
		case "/v1/workspaces/42/dataset_snapshots/snap-1":
			cmdtest.JSON(w, 200, map[string]interface{}{
				"id": "snap-1", "dataset_id": "ds-1",
				"columns": []map[string]interface{}{{"key": "k", "dbName": "col_0", "schema": map[string]string{"type": "string"}}},
			})
		default:
			t.Errorf("unexpected path %s", r.URL.Path)
		}
	})

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "snapshots", "list"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "snap-1") || !strings.Contains(env.Out.String(), "Total: 1") {
		t.Errorf("unexpected list output:\n%s", env.Out.String())
	}

	env.Out.Reset()
	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "snapshots", "get", "snap-1"); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "col_0") {
		t.Errorf("unexpected get output:\n%s", env.Out.String())
	}
}
