package integration_test

import (
	"bytes"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/aeltai/odyssey/internal/engine"
	"github.com/aeltai/odyssey/internal/grafana"
	"github.com/aeltai/odyssey/internal/sts"
)

// dashboardSpec defines expected results for a dashboard.
// MinPanels is the minimum number of panels we expect to parse.
// MaxWarnings is the maximum parse warnings we tolerate.
type dashboardSpec struct {
	MinPanels   int
	MaxWarnings int
}

// Top 47 Grafana dashboards by popularity, downloaded from grafana.com.
// These are the ground truth for regression testing.
var dashboards = map[string]dashboardSpec{
	"alertmanager":                    {MinPanels: 3, MaxWarnings: 0},
	"blackbox-exporter":               {MinPanels: 15, MaxWarnings: 0},
	"cadvisor":                        {MinPanels: 20, MaxWarnings: 5},
	"cassandra":                       {MinPanels: 1, MaxWarnings: 40},
	"ceph-cluster":                    {MinPanels: 10, MaxWarnings: 0},
	"cert-manager":                    {MinPanels: 20, MaxWarnings: 0},
	"coredns":                         {MinPanels: 10, MaxWarnings: 0},
	"docker-container":                {MinPanels: 40, MaxWarnings: 0},
	"envoy":                           {MinPanels: 10, MaxWarnings: 0},
	"etcd":                            {MinPanels: 25, MaxWarnings: 0},
	"grafana-internals":               {MinPanels: 25, MaxWarnings: 0},
	"haproxy":                         {MinPanels: 30, MaxWarnings: 10},
	"jenkins-performance":             {MinPanels: 30, MaxWarnings: 0},
	"jvm-jolokia":                     {MinPanels: 15, MaxWarnings: 0},
	"jvm-micrometer":                  {MinPanels: 12, MaxWarnings: 0},
	"kafka-exporter":                  {MinPanels: 5, MaxWarnings: 0},
	"kube-scheduler":                  {MinPanels: 5, MaxWarnings: 0},
	"kube-state-metrics":              {MinPanels: 30, MaxWarnings: 0},
	"kubernetes-apiserver":            {MinPanels: 10, MaxWarnings: 0},
	"kubernetes-calico":               {MinPanels: 0, MaxWarnings: 0},
	"kubernetes-cluster-autoscaler":   {MinPanels: 10, MaxWarnings: 0},
	"kubernetes-cluster-prometheus":   {MinPanels: 10, MaxWarnings: 0},
	"kubernetes-cluster":              {MinPanels: 30, MaxWarnings: 0},
	"kubernetes-deployment":           {MinPanels: 15, MaxWarnings: 0},
	"kubernetes-networking":           {MinPanels: 30, MaxWarnings: 0},
	"kubernetes-node-exporter":        {MinPanels: 30, MaxWarnings: 0},
	"kubernetes-pvc":                  {MinPanels: 30, MaxWarnings: 0},
	"kubewarden":                      {MinPanels: 20, MaxWarnings: 0},
	"loki-dashboard":                  {MinPanels: 10, MaxWarnings: 5},
	"memcached":                       {MinPanels: 40, MaxWarnings: 0},
	"minio":                           {MinPanels: 40, MaxWarnings: 0},
	"mongodb":                         {MinPanels: 15, MaxWarnings: 0},
	"mysql-overview":                  {MinPanels: 25, MaxWarnings: 0},
	"mysql-performance":               {MinPanels: 80, MaxWarnings: 0},
	"nginx-ingress-controller-nextgen": {MinPanels: 80, MaxWarnings: 0},
	"nginx-ingress":                   {MinPanels: 8, MaxWarnings: 0},
	"node-exporter-full":              {MinPanels: 200, MaxWarnings: 0},
	"node-exporter-server":            {MinPanels: 40, MaxWarnings: 0},
	"postgresql":                      {MinPanels: 35, MaxWarnings: 0},
	"prometheus":                      {MinPanels: 10, MaxWarnings: 0},
	"prometheus-node":                 {MinPanels: 5, MaxWarnings: 0},
	"prometheus-stats":                {MinPanels: 60, MaxWarnings: 0},
	"rabbitmq":                        {MinPanels: 35, MaxWarnings: 0},
	"redis":                           {MinPanels: 15, MaxWarnings: 0},
	"redis-dashboard":                 {MinPanels: 15, MaxWarnings: 0},
	"traefik":                         {MinPanels: 60, MaxWarnings: 0},
	"zookeeper":                       {MinPanels: 20, MaxWarnings: 0},
}

func testdataDir() string {
	_, filename, _, _ := runtime.Caller(0)
	return filepath.Join(filepath.Dir(filename), "..", "testdata", "dashboards")
}

// TestDashboardParsing verifies that every dashboard JSON can be parsed
// without crashing and produces the expected number of panels.
func TestDashboardParsing(t *testing.T) {
	dir := testdataDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("testdata/dashboards not found at %s — run 'make download-testdata'", dir)
	}

	for name, spec := range dashboards {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".json")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("%s not found", path)
			}

			title, panels, err := grafana.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}
			_ = title

			if len(panels) < spec.MinPanels {
				t.Errorf("parsed %d panels, want >= %d", len(panels), spec.MinPanels)
			}
		})
	}
}

// TestDashboardSanitization verifies that PromQL sanitisation doesn't
// introduce regressions (no new parse errors beyond known limits).
func TestDashboardSanitization(t *testing.T) {
	dir := testdataDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("testdata/dashboards not found at %s", dir)
	}

	for name, spec := range dashboards {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".json")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("%s not found", path)
			}

			_, panels, err := grafana.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile failed: %v", err)
			}

			var buf bytes.Buffer
			enriched := engine.SanitiseAndExtract(panels, "5m", nil, &buf)

			warnings := strings.Count(buf.String(), "warn:")
			if warnings > spec.MaxWarnings {
				t.Errorf("got %d PromQL warnings, max allowed %d\n%s",
					warnings, spec.MaxWarnings, buf.String())
			}

			// Every enriched panel should have either metrics or a parse error
			for _, ep := range enriched {
				if !ep.ParseError && ep.Sanitized == "" && len(ep.MetricNames) == 0 {
					// An empty sanitized expr with no error and no metrics
					// means we lost the expression during sanitization
					if ep.Panel.Expr != "" {
						t.Errorf("panel %q: expr %q was sanitized to empty without error",
							ep.Panel.Title, ep.Panel.Expr)
					}
				}
			}
		})
	}
}

// TestDashboardYAMLGeneration verifies that the full pipeline
// (parse → sanitise → build YAML) produces valid output without panics.
func TestDashboardYAMLGeneration(t *testing.T) {
	dir := testdataDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("testdata/dashboards not found at %s", dir)
	}

	for name := range dashboards {
		t.Run(name, func(t *testing.T) {
			path := filepath.Join(dir, name+".json")
			if _, err := os.Stat(path); os.IsNotExist(err) {
				t.Skipf("%s not found", path)
			}

			title, panels, err := grafana.ParseFile(path)
			if err != nil {
				t.Fatalf("ParseFile: %v", err)
			}

			var buf bytes.Buffer
			enriched := engine.SanitiseAndExtract(panels, "5m", nil, &buf)

			var inputs []sts.PanelInput
			for _, ep := range enriched {
				if !ep.ParseError {
					inputs = append(inputs, sts.PanelInput{
						Title: ep.Panel.Title,
						Expr:  ep.Sanitized,
					})
				}
			}

			if len(inputs) == 0 {
				return
			}

			dashName := title
			if dashName == "" {
				dashName = name
			}
			dash := sts.BuildDashboard(dashName, "publicDashboard", 0, inputs)

			outPath := filepath.Join(t.TempDir(), name+".sts.yaml")
			if err := sts.WriteDashboardYAML(dash, outPath); err != nil {
				t.Fatalf("WriteDashboardYAML: %v", err)
			}

			data, err := os.ReadFile(outPath)
			if err != nil {
				t.Fatalf("read output: %v", err)
			}

			content := string(data)
			if !strings.Contains(content, "name: "+dashName) {
				t.Error("YAML missing dashboard name")
			}
			if !strings.Contains(content, "kind: Grid") {
				t.Error("YAML missing Grid layout")
			}
			if !strings.Contains(content, "kind: PrometheusTimeSeriesQuery") {
				t.Error("YAML missing Prometheus query kind")
			}
		})
	}
}

// TestAllDashboardsNoCrash is a simple smoke test that ensures no
// dashboard JSON causes a panic or crash in the full pipeline.
func TestAllDashboardsNoCrash(t *testing.T) {
	dir := testdataDir()
	if _, err := os.Stat(dir); os.IsNotExist(err) {
		t.Skipf("testdata/dashboards not found at %s", dir)
	}

	files, err := filepath.Glob(filepath.Join(dir, "*.json"))
	if err != nil {
		t.Fatal(err)
	}

	totalPanels := 0
	for _, f := range files {
		_, panels, err := grafana.ParseFile(f)
		if err != nil {
			t.Errorf("%s: parse error: %v", filepath.Base(f), err)
			continue
		}
		totalPanels += len(panels)

		var buf bytes.Buffer
		engine.SanitiseAndExtract(panels, "5m", nil, &buf)
	}

	t.Logf("Processed %d dashboards, %d total panels — no crashes", len(files), totalPanels)
}
