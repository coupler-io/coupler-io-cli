package datasets_test

import (
	"net/http"
	"reflect"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdtest"
	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
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

func TestCreateCustomSQL(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost || r.URL.Path != "/v1/workspaces/42/dataflows/df-1/datasets" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{"type": "customSql", "query": `select * from "t"`, "name": "Summary"}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 201, map[string]interface{}{"id": "ds-2", "name": "Summary"})
	})

	err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "create", "--dataflow-id", "df-1",
		"--name", "Summary", "--query", `select * from "t"`)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "ds-2") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestUpdateClearsWithNull(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/workspaces/42/datasets/ds-1" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{"name": "New", "ai_context": nil}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"id": "ds-1", "name": "New"})
	})

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "update", "ds-1", "--name", "New", "--ai-context", ""); err != nil {
		t.Fatal(err)
	}
}

func TestUpdateQuerySetsCustomSQLType(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		want := map[string]interface{}{"type": "customSql", "query": "select 1"}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"id": "ds-1"})
	})

	if err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "update", "ds-1", "--query", "select 1"); err != nil {
		t.Fatal(err)
	}

	err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "update", "ds-1")
	if err == nil || !strings.Contains(err.Error(), "nothing to update") {
		t.Fatalf("got %v", err)
	}
	if code := cmdutil.ExitCode(err); code != cmdutil.ExitUsage {
		t.Errorf("exit code = %d, want %d", code, cmdutil.ExitUsage)
	}
}

func TestColumnsUpdateSingle(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPatch || r.URL.Path != "/v1/workspaces/42/datasets/ds-1/columns" {
			t.Errorf("unexpected request %s %s", r.Method, r.URL.Path)
		}
		want := map[string]interface{}{"column_updates": []interface{}{
			map[string]interface{}{"key": "revenue", "label": "Revenue", "description": nil},
		}}
		if body := cmdtest.DecodeBody(t, r); !reflect.DeepEqual(body, want) {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"id": "ds-1", "updated_column_keys": []string{"revenue"}})
	})

	err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "columns", "update", "ds-1",
		"--key", "revenue", "--label", "Revenue", "--description", "")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(env.Out.String(), "revenue") {
		t.Errorf("unexpected output:\n%s", env.Out.String())
	}
}

func TestColumnsUpdateBulkAndValidation(t *testing.T) {
	env := cmdtest.Setup(t, func(w http.ResponseWriter, r *http.Request) {
		body := cmdtest.DecodeBody(t, r)
		if updates, _ := body["column_updates"].([]interface{}); len(updates) != 2 {
			t.Errorf("body = %v", body)
		}
		cmdtest.JSON(w, 200, map[string]interface{}{"id": "ds-1", "updated_column_keys": []string{"a", "b"}})
	})

	err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), "columns", "update", "ds-1",
		"--updates", `[{"key":"a","label":"A"},{"key":"b","description":"B"}]`)
	if err != nil {
		t.Fatal(err)
	}

	for name, args := range map[string][]string{
		"no input":     {"columns", "update", "ds-1"},
		"key only":     {"columns", "update", "ds-1", "--key", "a"},
		"mixed inputs": {"columns", "update", "ds-1", "--key", "a", "--label", "A", "--updates", "[]"},
		"bad json":     {"columns", "update", "ds-1", "--updates", "{"},
	} {
		err := cmdtest.Run(datasets.NewCmdDatasets(env.Factory), args...)
		if err == nil {
			t.Errorf("%s: expected error", name)
		} else if code := cmdutil.ExitCode(err); code != cmdutil.ExitUsage {
			t.Errorf("%s: exit code = %d, want %d (usage error)", name, code, cmdutil.ExitUsage)
		}
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
