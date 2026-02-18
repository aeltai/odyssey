package grafana

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseFile_Targets(t *testing.T) {
	j := `{
		"title": "Test Dashboard",
		"panels": [
			{
				"title": "CPU",
				"type": "graph",
				"targets": [
					{"expr": "rate(cpu_total[5m])"},
					{"expr": "rate(cpu_user[5m])"}
				]
			},
			{
				"title": "Memory",
				"type": "graph",
				"targets": [
					{"expr": "node_memory_total"}
				]
			}
		]
	}`
	path := writeTmp(t, j)
	title, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Test Dashboard" {
		t.Errorf("title = %q, want Test Dashboard", title)
	}
	if len(panels) != 3 {
		t.Fatalf("got %d panels, want 3", len(panels))
	}
	if panels[0].Expr != "rate(cpu_total[5m])" {
		t.Errorf("panels[0].Expr = %q", panels[0].Expr)
	}
}

func TestParseFile_Rows(t *testing.T) {
	j := `{
		"title": "Row Dashboard",
		"rows": [
			{
				"title": "Network",
				"panels": [
					{
						"title": "Traffic",
						"type": "graph",
						"targets": [{"expr": "rate(net_rx[5m])"}]
					}
				]
			}
		]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 1 {
		t.Fatalf("got %d panels, want 1", len(panels))
	}
	if panels[0].Title != "Traffic" {
		t.Errorf("title = %q, want Traffic", panels[0].Title)
	}
}

func TestParseFile_NestedPanelsInRow(t *testing.T) {
	j := `{
		"title": "Nested",
		"panels": [
			{
				"type": "row",
				"title": "Row1",
				"panels": [
					{
						"title": "Inner",
						"type": "graph",
						"targets": [{"expr": "up"}]
					}
				]
			}
		]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 1 {
		t.Fatalf("got %d panels, want 1", len(panels))
	}
	if panels[0].Title != "Inner" {
		t.Errorf("title = %q, want Inner", panels[0].Title)
	}
}

func TestParseFile_OptionsQueries(t *testing.T) {
	j := `{
		"title": "OptDash",
		"panels": [
			{
				"title": "Opt",
				"type": "timeseries",
				"options": {
					"queries": [{"expr": "rate(http_requests[5m])"}]
				}
			}
		]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 1 {
		t.Fatalf("got %d panels, want 1", len(panels))
	}
	if panels[0].Expr != "rate(http_requests[5m])" {
		t.Errorf("Expr = %q", panels[0].Expr)
	}
}

func TestParseFile_QueryField(t *testing.T) {
	j := `{
		"title": "QDash",
		"panels": [
			{
				"title": "Q",
				"type": "table",
				"targets": [{"query": "sum(up)"}]
			}
		]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 1 {
		t.Fatalf("got %d, want 1", len(panels))
	}
	if panels[0].Expr != "sum(up)" {
		t.Errorf("Expr = %q", panels[0].Expr)
	}
}

func TestParseFile_DatasourceFilter(t *testing.T) {
	j := `{
		"title": "Mixed",
		"panels": [
			{
				"title": "Prom",
				"type": "timeseries",
				"targets": [{"expr": "rate(up[5m])"}]
			},
			{
				"title": "Loki",
				"type": "timeseries",
				"targets": [{"expr": "invalid", "datasource": {"type": "loki", "uid": "l1"}}]
			},
			{
				"title": "PromExplicit",
				"type": "timeseries",
				"targets": [{"expr": "sum(up)", "datasource": {"type": "prometheus", "uid": "p1"}}]
			}
		]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	// Only Prom and PromExplicit — Loki target skipped
	if len(panels) != 2 {
		t.Fatalf("got %d panels, want 2 (Loki should be excluded)", len(panels))
	}
	exprs := []string{panels[0].Expr, panels[1].Expr}
	if !contains(exprs, "rate(up[5m])") || !contains(exprs, "sum(up)") {
		t.Errorf("unexpected exprs: %v", exprs)
	}
}

func contains(s []string, x string) bool {
	for _, v := range s {
		if v == x {
			return true
		}
	}
	return false
}

func TestParseFile_EmptyPanels(t *testing.T) {
	j := `{"title": "Empty", "panels": []}`
	path := writeTmp(t, j)
	title, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if title != "Empty" {
		t.Errorf("title = %q", title)
	}
	if len(panels) != 0 {
		t.Errorf("got %d panels, want 0", len(panels))
	}
}

func TestParseFile_NoPanelsNoRows(t *testing.T) {
	j := `{"title": "Bare"}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 0 {
		t.Errorf("got %d panels, want 0", len(panels))
	}
}

func TestParseFile_DedupExpressions(t *testing.T) {
	j := `{
		"title": "Dedup",
		"panels": [
			{
				"title": "A",
				"type": "graph",
				"targets": [{"expr": "up"}, {"expr": "up"}]
			}
		]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 1 {
		t.Errorf("duplicate expr should be deduped: got %d, want 1", len(panels))
	}
}

func TestParseFile_TitleInheritance(t *testing.T) {
	j := `{
		"title": "Inherit",
		"panels": [
			{
				"type": "row",
				"title": "RowTitle",
				"panels": [
					{"type": "graph", "targets": [{"expr": "up"}]}
				]
			}
		]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if len(panels) != 1 {
		t.Fatalf("got %d", len(panels))
	}
	if panels[0].Title != "RowTitle" {
		t.Errorf("expected inherited title 'RowTitle', got %q", panels[0].Title)
	}
}

func TestParseFile_InvalidJSON(t *testing.T) {
	path := writeTmp(t, `not json at all`)
	_, _, err := ParseFile(path)
	if err == nil {
		t.Error("expected error for invalid JSON")
	}
}

func TestParseFile_FileNotFound(t *testing.T) {
	_, _, err := ParseFile("/nonexistent/file.json")
	if err == nil {
		t.Error("expected error for missing file")
	}
}

func TestParseFile_SourceFieldPopulated(t *testing.T) {
	j := `{
		"title": "Src",
		"panels": [{"title":"A","type":"graph","targets":[{"expr":"up"}]}]
	}`
	path := writeTmp(t, j)
	_, panels, err := ParseFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if panels[0].Source != path {
		t.Errorf("Source = %q, want %q", panels[0].Source, path)
	}
}

func TestParseFileResult_VariableDefaults(t *testing.T) {
	j := `{
		"title": "Vars",
		"panels": [{"title":"P","type":"graph","targets":[{"expr":"up"}]}],
		"templating": {
			"list": [
				{"name": "namespace", "type": "query", "current": {"text": "prod", "value": "prod"}},
				{"name": "job", "type": "query", "current": {"text": "myjob", "value": "myjob"}},
				{"name": "ds", "type": "datasource", "current": {"text": "Prometheus", "value": "Prometheus"}},
				{"name": "__interval", "type": "interval", "current": {"text": "5m", "value": "5m"}},
				{"name": "empty_var", "type": "query", "current": {}},
				{"name": "all_var", "type": "query", "current": {"text": "All", "value": "$__all"}}
			]
		}
	}`
	path := writeTmp(t, j)
	result, err := ParseFileResult(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "Vars" {
		t.Errorf("Title = %q", result.Title)
	}
	if len(result.Panels) != 1 {
		t.Fatalf("got %d panels", len(result.Panels))
	}
	if result.VariableDefaults == nil {
		t.Fatal("VariableDefaults is nil")
	}
	if v := result.VariableDefaults["namespace"]; v != "prod" {
		t.Errorf("namespace = %q, want prod", v)
	}
	if v := result.VariableDefaults["job"]; v != "myjob" {
		t.Errorf("job = %q, want myjob", v)
	}
	if _, ok := result.VariableDefaults["ds"]; ok {
		t.Error("datasource variable should be skipped")
	}
	if _, ok := result.VariableDefaults["__interval"]; ok {
		t.Error("__interval should be skipped")
	}
	if _, ok := result.VariableDefaults["empty_var"]; ok {
		t.Error("empty_var should be skipped")
	}
	if _, ok := result.VariableDefaults["all_var"]; ok {
		t.Error("$__all value should be skipped")
	}
}

func TestParseFileResult_VariableArrayValue(t *testing.T) {
	j := `{
		"title": "ArrayVal",
		"panels": [{"title":"P","type":"graph","targets":[{"expr":"up"}]}],
		"templating": {
			"list": [
				{"name": "instance", "type": "query", "current": {"text": "host1", "value": ["host1", "host2"]}}
			]
		}
	}`
	path := writeTmp(t, j)
	result, err := ParseFileResult(path)
	if err != nil {
		t.Fatal(err)
	}
	if v := result.VariableDefaults["instance"]; v != "host1" {
		t.Errorf("instance = %q, want host1 (first element)", v)
	}
}

func TestParseFileResult_NoTemplating(t *testing.T) {
	j := `{"title": "Plain", "panels": [{"title":"P","type":"graph","targets":[{"expr":"up"}]}]}`
	path := writeTmp(t, j)
	result, err := ParseFileResult(path)
	if err != nil {
		t.Fatal(err)
	}
	if result.VariableDefaults != nil {
		t.Errorf("expected nil VariableDefaults, got %v", result.VariableDefaults)
	}
}

func TestParseBytesResult(t *testing.T) {
	j := `{
		"title": "BytesParse",
		"panels": [{"title":"X","type":"graph","targets":[{"expr":"up"}]}],
		"templating": {"list": [{"name": "env", "type": "custom", "current": {"value": "staging"}}]}
	}`
	result, err := ParseBytesResult([]byte(j))
	if err != nil {
		t.Fatal(err)
	}
	if result.Title != "BytesParse" {
		t.Errorf("Title = %q", result.Title)
	}
	if v := result.VariableDefaults["env"]; v != "staging" {
		t.Errorf("env = %q, want staging", v)
	}
}

func writeTmp(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "test.json")
	if err := os.WriteFile(path, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	return path
}
