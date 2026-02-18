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

// GrafanaVariable holds the full metadata for a Grafana template variable.
type GrafanaVariable struct {
	Name       string
	Type       string   // query, custom, textbox, constant, interval, datasource
	Label      string   // display label
	Query      string   // for query type: label_values(metric, label) etc.
	Current    string   // current/default value
	Multi      bool     // allow multiple values
	IncludeAll bool     // include "All" option
	AllValue   string   // custom all value (e.g. ".*")
	Options    []string // for custom/interval: the option values
	Sort       int      // sort order (0=disabled, 1=alpha-asc, 2=alpha-desc, 3=num-asc, 4=num-desc)
}

// ParseResult holds everything extracted from a Grafana dashboard JSON.
type ParseResult struct {
	Title            string
	Panels           []Panel
	Variables        []GrafanaVariable // full variable definitions from templating.list
	VariableDefaults map[string]string // variable name -> current/default value from templating.list
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
	Title      string        `json:"title"`
	Panels     []rawPanel    `json:"panels"`
	Rows       []rawRow      `json:"rows"`
	Templating rawTemplating `json:"templating"`
}

type rawTemplating struct {
	List []rawVariable `json:"list"`
}

type rawVariable struct {
	Name       string          `json:"name"`
	Type       string          `json:"type"`
	Label      string          `json:"label"`
	Query      interface{}     `json:"query"`
	Current    json.RawMessage `json:"current"`
	Multi      bool            `json:"multi"`
	IncludeAll bool            `json:"includeAll"`
	AllValue   string          `json:"allValue"`
	Options    []rawOption     `json:"options"`
	Sort       int             `json:"sort"`
}

type rawOption struct {
	Text     interface{} `json:"text"`
	Value    interface{} `json:"value"`
	Selected bool        `json:"selected"`
}

// ParseFile reads a Grafana dashboard JSON and returns the dashboard title
// together with all panels that contain PromQL expressions.
func ParseFile(path string) (string, []Panel, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return "", nil, fmt.Errorf("read %s: %w", path, err)
	}
	r, err := parseRawResult(data, path)
	if err != nil {
		return "", nil, err
	}
	return r.Title, r.Panels, nil
}

// ParseBytes parses a Grafana dashboard from raw JSON bytes.
func ParseBytes(data []byte) (string, []Panel, error) {
	r, err := parseRawResult(data, "upload")
	if err != nil {
		return "", nil, err
	}
	return r.Title, r.Panels, nil
}

// ParseFileResult is like ParseFile but returns the full ParseResult including variable defaults.
func ParseFileResult(path string) (*ParseResult, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", path, err)
	}
	return parseRawResult(data, path)
}

// ParseBytesResult is like ParseBytes but returns the full ParseResult including variable defaults.
func ParseBytesResult(data []byte) (*ParseResult, error) {
	return parseRawResult(data, "upload")
}

func parseRawResult(data []byte, source string) (*ParseResult, error) {
	var dash rawDashboard
	if err := json.Unmarshal(data, &dash); err != nil {
		return nil, fmt.Errorf("parse dashboard JSON: %w", err)
	}
	var panels []Panel
	walkPanels(dash.Panels, "", source, &panels)
	for _, row := range dash.Rows {
		walkPanels(row.Panels, strings.TrimSpace(row.Title), source, &panels)
	}
	return &ParseResult{
		Title:            dash.Title,
		Panels:           panels,
		Variables:        extractVariables(dash),
		VariableDefaults: extractVariableDefaults(dash),
	}, nil
}

func extractVariables(dash rawDashboard) []GrafanaVariable {
	var vars []GrafanaVariable
	for _, v := range dash.Templating.List {
		if v.Type == "datasource" || strings.HasPrefix(v.Name, "__") {
			continue
		}
		gv := GrafanaVariable{
			Name:       v.Name,
			Type:       v.Type,
			Label:      v.Label,
			Multi:      v.Multi,
			IncludeAll: v.IncludeAll,
			AllValue:   v.AllValue,
			Sort:       v.Sort,
			Current:    parseCurrentValue(v.Current),
		}
		switch q := v.Query.(type) {
		case string:
			gv.Query = q
		case map[string]interface{}:
			if s, ok := q["query"].(string); ok {
				gv.Query = s
			}
		}
		for _, opt := range v.Options {
			if s, ok := opt.Value.(string); ok && s != "" && s != "$__all" && !strings.HasPrefix(s, "$__auto") {
				gv.Options = append(gv.Options, s)
			}
		}
		vars = append(vars, gv)
	}
	return vars
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

// extractVariableDefaults parses templating.list and returns a map of variable
// name to its current/default value. Built-in variables (datasource, __*) and
// variables without a usable current value are skipped.
func extractVariableDefaults(dash rawDashboard) map[string]string {
	result := map[string]string{}
	for _, v := range dash.Templating.List {
		if v.Type == "datasource" || strings.HasPrefix(v.Name, "__") {
			continue
		}
		if len(v.Current) == 0 {
			continue
		}
		val := parseCurrentValue(v.Current)
		if val == "" || val == "$__all" || strings.HasPrefix(val, "$__auto") {
			continue
		}
		result[v.Name] = val
	}
	if len(result) == 0 {
		return nil
	}
	return result
}

func parseCurrentValue(raw json.RawMessage) string {
	var obj struct {
		Value interface{} `json:"value"`
		Text  interface{} `json:"text"`
	}
	if err := json.Unmarshal(raw, &obj); err != nil {
		return ""
	}
	if s, ok := obj.Value.(string); ok && s != "" {
		return s
	}
	if arr, ok := obj.Value.([]interface{}); ok && len(arr) > 0 {
		if s, ok := arr[0].(string); ok {
			return s
		}
	}
	if s, ok := obj.Text.(string); ok && s != "" {
		return s
	}
	return ""
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
