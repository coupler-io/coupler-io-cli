package cmdutil_test

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/coupler-io/coupler-io-cli/internal/cmdutil"
)

func newFactory(stdin string) *cmdutil.Factory {
	return &cmdutil.Factory{IOStreams: &cmdutil.IOStreams{In: strings.NewReader(stdin)}}
}

func TestParseParams(t *testing.T) {
	f := newFactory("")

	got, err := f.ParseParams(`{"a":1,"b":"x"}`, []string{"b=y", "c=d=e"})
	if err != nil {
		t.Fatal(err)
	}
	want := map[string]interface{}{"a": float64(1), "b": "y", "c": "d=e"}
	if !reflect.DeepEqual(got, want) {
		t.Errorf("got %v, want %v", got, want)
	}

	if got, err := f.ParseParams("", nil); err != nil || got != nil {
		t.Errorf("no input should give nil, got %v, %v", got, err)
	}
}

func TestParseParamsErrors(t *testing.T) {
	f := newFactory("")
	if _, err := f.ParseParams("", []string{"novalue"}); err == nil || !strings.Contains(err.Error(), "key=value") {
		t.Errorf("expected key=value error, got %v", err)
	}
	if _, err := f.ParseParams(`[1]`, nil); err == nil || !strings.Contains(err.Error(), "JSON object") {
		t.Errorf("expected JSON object error, got %v", err)
	}
}

func TestReadArg(t *testing.T) {
	path := filepath.Join(t.TempDir(), "q.sql")
	if err := os.WriteFile(path, []byte("select 1\n"), 0o600); err != nil {
		t.Fatal(err)
	}

	f := newFactory("from stdin\n")
	for in, want := range map[string]string{"plain": "plain", "@" + path: "select 1", "-": "from stdin"} {
		got, err := f.ReadArg(in)
		if err != nil || got != want {
			t.Errorf("ReadArg(%q) = %q, %v; want %q", in, got, err, want)
		}
	}
	if _, err := f.ReadArg("@/does/not/exist"); err == nil {
		t.Error("expected error for missing file")
	}
}

func TestPath(t *testing.T) {
	if got := cmdutil.Path("workspaces", "42", "datasets", "a/b c"); got != "/v1/workspaces/42/datasets/a%2Fb%20c" {
		t.Errorf("Path = %q", got)
	}
}
