package grafana

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// Panel represents a single Grafana panel with its title and PromQL expression.
type Panel struct {
	Title  string
	Expr   string
	Source string
}

type rawPanel struct {
	Type    string      `json:"type"`
	Title   string      `json:"title"`
	Targets []rawTarget `json:"targets"`
	Panels  []rawPanel  `json:"panels"`
	Options *rawOptions `json:"options"`
}

type rawTarget struct {
	Expr       string          `json:"expr"`
	Query      string          `json:"query"`
	Datasource json.RawMessage `json:"datasource"`
}

type rawOptions struct {
	Queries []rawTarget `json:"queries"`
}

type rawRow struct {
	Title  string     `json:"title"`
	Panels []rawPanel `json:"panels"`
}

type rawDashboard struct {
	Title  string     `json:"title"`
	Panels []rawPanel `json:"panels"`
	Rows   []rawRow   `json:"rows"`
}

// ParseFile reads a Grafana dashboard JSON and returns the dashboard title
// together with all panels that contain PromQL expressions.
func ParseFile(path string) (string, []Panel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read %s: %w", path, err)
	}
	return parseRaw(data, path)
}

// ParseBytes parses a Grafana dashboard from raw JSON bytes.
func ParseBytes(data []byte) (string, []Panel, error) {
	return parseRaw(data, "upload")
}

func parseRaw(data []byte, source string) (string, []Panel, error) {
	var dash rawDashboard
	if err := json.Unmarshal(data, &dash); err != nil {
		return "", nil, fmt.Errorf("parse dashboard JSON: %w", err)
	}
	var panels []Panel
	walkPanels(dash.Panels, "", source, &panels)
	for _, row := range dash.Rows {
		walkPanels(row.Panels, strings.TrimSpace(row.Title), source, &panels)
	}
	return dash.Title, panels, nil
}

func walkPanels(raw []rawPanel, parentTitle, source string, out *[]Panel) {
	for i, p := range raw {
		title := strings.TrimSpace(p.Title)
		if title == "" {
			title = parentTitle
		}

		if p.Type == "row" {
			if len(p.Panels) > 0 {
				walkPanels(p.Panels, title, source, out)
			}
			continue
		}

		if len(p.Panels) > 0 {
			walkPanels(p.Panels, title, source, out)
		}

		exprs := extractExprs(p)
		for _, expr := range exprs {
			name := title
			if name == "" {
				name = fmt.Sprintf("Panel %d", i+1)
			}
			*out = append(*out, Panel{Title: name, Expr: expr, Source: source})
		}
	}
}

func extractExprs(p rawPanel) []string {
	var exprs []string
	seen := map[string]bool{}

	add := func(s string) {
		s = strings.TrimSpace(s)
		if s != "" && !seen[s] {
			seen[s] = true
			exprs = append(exprs, s)
		}
	}

	for _, t := range p.Targets {
		if !isPrometheusDatasource(t.Datasource) {
			continue
		}
		add(t.Expr)
		if t.Expr == "" {
			add(t.Query)
		}
	}

	if p.Options != nil {
		for _, q := range p.Options.Queries {
			add(q.Expr)
			if q.Expr == "" {
				add(q.Query)
			}
		}
	}

	return exprs
}

// isPrometheusDatasource returns false if datasource is explicitly non-Prometheus.
// Returns true for null, empty, or Prometheus (incl. by uid when type unknown).
func isPrometheusDatasource(ds json.RawMessage) bool {
	if len(ds) == 0 {
		return true // null or omitted — use default (often Prometheus)
	}
	// Try to parse as object: {"type":"prometheus",...} or {"type":"loki",...}
	var m map[string]interface{}
	if err := json.Unmarshal(ds, &m); err != nil {
		// String datasource like "Prometheus" or uid "abc123"
		var s string
		if err := json.Unmarshal(ds, &s); err != nil {
			return true // Unknown format — include to be safe
		}
		return s == "" || strings.EqualFold(s, "prometheus")
	}
	t, _ := m["type"].(string)
	if t == "" {
		return true
	}
	return strings.EqualFold(t, "prometheus")
}
