package sts

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestBuildDashboard_Basic(t *testing.T) {
	inputs := []PanelInput{
		{Title: "CPU", Expr: `rate(cpu[5m])`},
		{Title: "Memory", Expr: `node_mem_total`},
	}
	d := BuildDashboard("Test", "publicDashboard", 0, inputs)

	if d.Name != "Test" {
		t.Errorf("Name = %q", d.Name)
	}
	if d.Scope != "publicDashboard" {
		t.Errorf("Scope = %q", d.Scope)
	}
	if len(d.Dashboard.Spec.Panels) != 2 {
		t.Errorf("got %d panels, want 2", len(d.Dashboard.Spec.Panels))
	}
	if len(d.Dashboard.Spec.Layouts) != 1 {
		t.Errorf("got %d layouts, want 1", len(d.Dashboard.Spec.Layouts))
	}
	items := d.Dashboard.Spec.Layouts[0].Spec.Items
	if len(items) != 2 {
		t.Fatalf("got %d items, want 2", len(items))
	}
	for _, item := range items {
		if !strings.HasPrefix(item.Content.Ref, "#/spec/panels/") {
			t.Errorf("ref = %q", item.Content.Ref)
		}
	}
}

func TestBuildDashboard_WithID(t *testing.T) {
	d := BuildDashboard("X", "s", 42, []PanelInput{{Title: "A", Expr: "up"}})
	if d.ID != 42 {
		t.Errorf("ID = %d, want 42", d.ID)
	}
}

func TestBuildDashboard_GridLayout(t *testing.T) {
	inputs := make([]PanelInput, 7)
	for i := range inputs {
		inputs[i] = PanelInput{Title: "P", Expr: "up"}
	}
	d := BuildDashboard("Grid", "s", 0, inputs)
	items := d.Dashboard.Spec.Layouts[0].Spec.Items
	if items[0].X != 0 || items[0].Y != 0 {
		t.Errorf("item 0: (%d,%d), want (0,0)", items[0].X, items[0].Y)
	}
	if items[3].X != 0 || items[3].Y != 2 {
		t.Errorf("item 3: (%d,%d), want (0,2)", items[3].X, items[3].Y)
	}
}

func TestBuildDashboard_PanelQuery(t *testing.T) {
	d := BuildDashboard("Q", "s", 0, []PanelInput{{Title: "T", Expr: "rate(x[5m])"}})
	for _, p := range d.Dashboard.Spec.Panels {
		if len(p.Spec.Queries) != 1 {
			t.Fatalf("queries = %d", len(p.Spec.Queries))
		}
		q := p.Spec.Queries[0]
		if q.Kind != "TimeSeriesQuery" {
			t.Errorf("kind = %q", q.Kind)
		}
		if q.Spec.Plugin.Kind != "PrometheusTimeSeriesQuery" {
			t.Errorf("plugin = %q", q.Spec.Plugin.Kind)
		}
		if q.Spec.Plugin.Spec.Query != "rate(x[5m])" {
			t.Errorf("query = %q", q.Spec.Plugin.Spec.Query)
		}
	}
}

func TestWriteDashboardYAML(t *testing.T) {
	d := BuildDashboard("WriteTest", "publicDashboard", 0, []PanelInput{{Title: "A", Expr: "up"}})
	path := filepath.Join(t.TempDir(), "out.yaml")

	if err := WriteDashboardYAML(d, path); err != nil {
		t.Fatal(err)
	}
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	content := string(data)
	for _, want := range []string{"name: WriteTest", "scope: publicDashboard", "query: up", "kind: Grid"} {
		if !strings.Contains(content, want) {
			t.Errorf("YAML missing %q", want)
		}
	}
}
