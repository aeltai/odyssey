package sts

import (
	"crypto/sha256"
	"fmt"
	"os"
	"strings"

	"gopkg.in/yaml.v3"
)

// PanelInput is the data needed to generate one STS panel.
type PanelInput struct {
	Title string
	Expr  string
}

// Dashboard is the top-level STS dashboard YAML structure.
type Dashboard struct {
	ID          int64              `yaml:"id,omitempty"`
	Name        string             `yaml:"name"`
	Description string             `yaml:"description"`
	Scope       string             `yaml:"scope"`
	Dashboard   DashboardContainer `yaml:"dashboard"`
}

type DashboardContainer struct {
	Metadata DashMetadata `yaml:"metadata"`
	Spec     DashSpec     `yaml:"spec"`
}

type DashMetadata struct {
	Project string `yaml:"project"`
}

type DashSpec struct {
	Variables []STSVariable    `yaml:"variables,omitempty"`
	Layouts   []Layout         `yaml:"layouts"`
	Panels    map[string]Panel `yaml:"panels"`
}

// STSVariable wraps the Perses-style variable union used by STS dashboards.
type STSVariable struct {
	PersesListVariable *PersesVariable `yaml:"perseslistvariable,omitempty"`
	PersesTextVariable *PersesVariable `yaml:"persestextvariable,omitempty"`
}

// PersesVariable is the shared structure for both list and text variables.
type PersesVariable struct {
	Kind string            `yaml:"kind"`
	Spec PersesVariableSpec `yaml:"spec"`
}

type PersesVariableSpec struct {
	Name          string          `yaml:"name"`
	Display       VariableDisplay `yaml:"display"`
	AllowAllValue bool            `yaml:"allowAllValue,omitempty"`
	AllowMultiple bool            `yaml:"allowMultiple,omitempty"`
	Sort          string          `yaml:"sort,omitempty"`
	Plugin        *VariablePlugin `yaml:"plugin,omitempty"`
	Value         string          `yaml:"value,omitempty"`
}

type VariableDisplay struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type VariablePlugin struct {
	Kind string             `yaml:"kind"`
	Spec VariablePluginSpec `yaml:"spec"`
}

type VariablePluginSpec struct {
	LabelName string   `yaml:"labelName,omitempty"`
	Matchers  []string `yaml:"matchers,omitempty"`
	Values    []string `yaml:"values,omitempty"`
}

type Layout struct {
	Kind string     `yaml:"kind"`
	Spec LayoutSpec `yaml:"spec"`
}

type LayoutSpec struct {
	Items []LayoutItem `yaml:"items"`
}

type LayoutItem struct {
	X       int        `yaml:"x"`
	Y       int        `yaml:"y"`
	Width   int        `yaml:"width"`
	Height  int        `yaml:"height"`
	Content ContentRef `yaml:"content"`
}

type ContentRef struct {
	Ref string `yaml:"$ref"`
}

type Panel struct {
	Spec PanelSpec `yaml:"spec"`
}

type PanelSpec struct {
	Display PanelDisplay `yaml:"display"`
	Plugin  PanelPlugin  `yaml:"plugin"`
	Queries []PanelQuery `yaml:"queries"`
}

type PanelDisplay struct {
	Name        string `yaml:"name"`
	Description string `yaml:"description"`
}

type PanelPlugin struct {
	Kind string         `yaml:"kind"`
	Spec TimeSeriesSpec `yaml:"spec"`
}

type TimeSeriesSpec struct {
	Legend     Legend     `yaml:"legend"`
	Thresholds Thresholds `yaml:"thresholds"`
	Visual    Visual     `yaml:"visual"`
	YAxis     YAxis      `yaml:"yAxis"`
}

type Legend struct {
	Mode     string   `yaml:"mode"`
	Position string   `yaml:"position"`
	Show     bool     `yaml:"show"`
	Values   []string `yaml:"values"`
}

type Thresholds struct {
	Mode  string        `yaml:"mode"`
	Steps []interface{} `yaml:"steps"`
}

type Visual struct {
	ConnectNulls bool `yaml:"connectNulls"`
}

type YAxis struct {
	Format YAxisFormat `yaml:"format"`
	Show   bool        `yaml:"show"`
}

type YAxisFormat struct {
	DecimalPlaces int `yaml:"decimalPlaces"`
}

type PanelQuery struct {
	Kind string         `yaml:"kind"`
	Spec PanelQuerySpec `yaml:"spec"`
}

type PanelQuerySpec struct {
	Plugin QueryPlugin `yaml:"plugin"`
}

type QueryPlugin struct {
	Kind string        `yaml:"kind"`
	Spec PromQuerySpec `yaml:"spec"`
}

type PromQuerySpec struct {
	Query string `yaml:"query"`
}

// VariableInput holds the Grafana variable metadata needed to build STS variable definitions.
type VariableInput struct {
	Name       string
	Type       string   // query, custom, textbox, constant, interval
	Label      string
	Query      string   // Grafana query string, e.g. label_values(metric, label)
	Multi      bool
	IncludeAll bool
	AllValue   string
	Options    []string // static option values
	Default    string   // current/default value
	Sort       int
}

// MapVariables converts Grafana variable definitions to STS dashboard variables.
func MapVariables(inputs []VariableInput) []STSVariable {
	var result []STSVariable
	for _, inp := range inputs {
		sv := mapVariable(inp)
		if sv != nil {
			result = append(result, *sv)
		}
	}
	return result
}

func mapVariable(inp VariableInput) *STSVariable {
	displayName := inp.Label
	if displayName == "" {
		displayName = inp.Name
	}

	switch inp.Type {
	case "query":
		return mapQueryVariable(inp, displayName)
	case "custom":
		return mapCustomVariable(inp, displayName)
	case "interval":
		return mapIntervalVariable(inp, displayName)
	case "textbox":
		return mapTextVariable(inp, displayName)
	case "constant":
		return &STSVariable{
			PersesTextVariable: &PersesVariable{
				Kind: "TextVariable",
				Spec: PersesVariableSpec{
					Name:    inp.Name,
					Display: VariableDisplay{Name: displayName},
					Value:   inp.Default,
				},
			},
		}
	default:
		return mapTextVariable(inp, displayName)
	}
}

func grafanaSortToSTS(sort int) string {
	switch sort {
	case 1:
		return "alphabetical-asc"
	case 2:
		return "alphabetical-desc"
	case 3:
		return "numerical-asc"
	case 4:
		return "numerical-desc"
	default:
		return "none"
	}
}

func mapQueryVariable(inp VariableInput, displayName string) *STSVariable {
	labelName, matchers := parseLabelValuesQuery(inp.Query)
	if labelName != "" {
		return &STSVariable{
			PersesListVariable: &PersesVariable{
				Kind: "ListVariable",
				Spec: PersesVariableSpec{
					Name:          inp.Name,
					Display:       VariableDisplay{Name: displayName},
					AllowAllValue: inp.IncludeAll,
					AllowMultiple: inp.Multi,
					Sort:          grafanaSortToSTS(inp.Sort),
					Plugin: &VariablePlugin{
						Kind: "MetricLabelValues",
						Spec: VariablePluginSpec{
							LabelName: labelName,
							Matchers:  matchers,
						},
					},
				},
			},
		}
	}

	// Handle query_result(...) — fall back to text variable since STS
	// doesn't have a direct equivalent. The user can set the value manually.
	return mapTextVariable(inp, displayName)
}

func mapCustomVariable(inp VariableInput, displayName string) *STSVariable {
	values := inp.Options
	if len(values) == 0 && inp.Query != "" {
		values = splitCustomValues(inp.Query)
	}
	if len(values) == 0 {
		return mapTextVariable(inp, displayName)
	}
	return &STSVariable{
		PersesListVariable: &PersesVariable{
			Kind: "ListVariable",
			Spec: PersesVariableSpec{
				Name:          inp.Name,
				Display:       VariableDisplay{Name: displayName},
				AllowAllValue: inp.IncludeAll,
				AllowMultiple: inp.Multi,
				Sort:          "none",
				Plugin: &VariablePlugin{
					Kind: "StaticListVariable",
					Spec: VariablePluginSpec{
						Values: values,
					},
				},
			},
		},
	}
}

func mapIntervalVariable(inp VariableInput, displayName string) *STSVariable {
	values := inp.Options
	if len(values) == 0 && inp.Query != "" {
		values = splitCustomValues(inp.Query)
	}
	if len(values) == 0 {
		values = []string{"1m", "5m", "15m", "30m", "1h", "6h", "12h", "1d"}
	}
	return &STSVariable{
		PersesListVariable: &PersesVariable{
			Kind: "ListVariable",
			Spec: PersesVariableSpec{
				Name:          inp.Name,
				Display:       VariableDisplay{Name: displayName},
				AllowAllValue: false,
				AllowMultiple: false,
				Sort:          "none",
				Plugin: &VariablePlugin{
					Kind: "StaticListVariable",
					Spec: VariablePluginSpec{
						Values: values,
					},
				},
			},
		},
	}
}

func mapTextVariable(inp VariableInput, displayName string) *STSVariable {
	return &STSVariable{
		PersesTextVariable: &PersesVariable{
			Kind: "TextVariable",
			Spec: PersesVariableSpec{
				Name:    inp.Name,
				Display: VariableDisplay{Name: displayName},
				Value:   inp.Default,
			},
		},
	}
}

// parseLabelValuesQuery parses Grafana's label_values(metric, label) query format.
// Returns (labelName, matchers). If the query doesn't match, returns ("", nil).
func parseLabelValuesQuery(query string) (string, []string) {
	query = strings.TrimSpace(query)
	if !strings.HasPrefix(query, "label_values(") || !strings.HasSuffix(query, ")") {
		return "", nil
	}
	inner := query[len("label_values(") : len(query)-1]
	inner = strings.TrimSpace(inner)

	// label_values(label_name) — single arg
	if !strings.Contains(inner, ",") {
		return strings.TrimSpace(inner), nil
	}

	// label_values(metric_expr, label_name) — two args
	lastComma := strings.LastIndex(inner, ",")
	metricExpr := strings.TrimSpace(inner[:lastComma])
	labelName := strings.TrimSpace(inner[lastComma+1:])

	// Strip any label filters from metric for the matcher: metric{foo="bar"} -> metric
	matcher := metricExpr
	if idx := strings.Index(matcher, "{"); idx > 0 {
		matcher = strings.TrimSpace(matcher[:idx])
	}

	var matchers []string
	if matcher != "" {
		matchers = []string{matcher}
	}
	return labelName, matchers
}

func splitCustomValues(query string) []string {
	parts := strings.Split(query, ",")
	var result []string
	for _, p := range parts {
		p = strings.TrimSpace(p)
		if p != "" {
			result = append(result, p)
		}
	}
	return result
}

func panelID(expr string, index int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", expr, index)))
	return fmt.Sprintf("%x", h[:6])[:11]
}

const gridCols = 3

// BuildDashboard creates a Dashboard from a list of panel inputs and optional variable definitions.
func BuildDashboard(name, scope string, dashID int64, inputs []PanelInput, variables ...[]STSVariable) Dashboard {
	panels := make(map[string]Panel, len(inputs))
	items := make([]LayoutItem, 0, len(inputs))

	for i, inp := range inputs {
		pid := panelID(inp.Expr, i)
		panels[pid] = Panel{
			Spec: PanelSpec{
				Display: PanelDisplay{Name: inp.Title},
				Plugin: PanelPlugin{
					Kind: "TimeSeriesChart",
					Spec: TimeSeriesSpec{
						Legend:     Legend{Mode: "list", Position: "right", Show: true, Values: []string{}},
						Thresholds: Thresholds{Mode: "absolute", Steps: []interface{}{}},
						Visual:    Visual{ConnectNulls: false},
						YAxis:     YAxis{Format: YAxisFormat{DecimalPlaces: 2}, Show: true},
					},
				},
				Queries: []PanelQuery{
					{
						Kind: "TimeSeriesQuery",
						Spec: PanelQuerySpec{
							Plugin: QueryPlugin{
								Kind: "PrometheusTimeSeriesQuery",
								Spec: PromQuerySpec{Query: inp.Expr},
							},
						},
					},
				},
			},
		}
		col := i % gridCols
		row := i / gridCols
		items = append(items, LayoutItem{
			X: col * 8, Y: row * 2, Width: 8, Height: 2,
			Content: ContentRef{Ref: "#/spec/panels/" + pid},
		})
	}

	var stsVars []STSVariable
	if len(variables) > 0 {
		stsVars = variables[0]
	}

	saveVars := "false"
	if len(stsVars) > 0 {
		saveVars = "true"
	}

	return Dashboard{
		ID:          dashID,
		Name:        name,
		Description: "Migrated from Grafana by Odyssey",
		Scope:       scope,
		Dashboard: DashboardContainer{
			Metadata: DashMetadata{Project: fmt.Sprintf(`{"saveVariables":%s}`, saveVars)},
			Spec: DashSpec{
				Variables: stsVars,
				Layouts: []Layout{
					{Kind: "Grid", Spec: LayoutSpec{Items: items}},
				},
				Panels: panels,
			},
		},
	}
}

// WriteDashboardYAML serialises the dashboard to a YAML file.
func WriteDashboardYAML(dash Dashboard, path string) error {
	f, err := os.Create(path)
	if err != nil {
		return err
	}
	defer f.Close()
	enc := yaml.NewEncoder(f)
	enc.SetIndent(2)
	return enc.Encode(dash)
}
