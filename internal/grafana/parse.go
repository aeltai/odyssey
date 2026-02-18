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
	Expr  string `json:"expr"`
	Query string `json:"query"`
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
	var dash rawDashboard
	if err := json.Unmarshal(data, &dash); err != nil {
		return "", nil, fmt.Errorf("parse %s: %w", path, err)
	}
	var panels []Panel
	walkPanels(dash.Panels, "", path, &panels)
	for _, row := range dash.Rows {
		walkPanels(row.Panels, strings.TrimSpace(row.Title), path, &panels)
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
