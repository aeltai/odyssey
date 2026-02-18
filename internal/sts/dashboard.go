package sts

import (
	"crypto/sha256"
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// PanelInput is the data needed to generate one STS panel.
type PanelInput struct {
	Title string
	Expr  string
}

// Dashboard is the top-level STS dashboard YAML structure.
type Dashboard struct {
	ID        int64              `yaml:"id,omitempty"`
	Name      string             `yaml:"name"`
	Scope     string             `yaml:"scope"`
	Dashboard DashboardContainer `yaml:"dashboard"`
}

type DashboardContainer struct {
	Metadata DashMetadata `yaml:"metadata"`
	Spec     DashSpec     `yaml:"spec"`
}

type DashMetadata struct {
	Project string `yaml:"project"`
}

type DashSpec struct {
	Layouts []Layout         `yaml:"layouts"`
	Panels  map[string]Panel `yaml:"panels"`
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

func panelID(expr string, index int) string {
	h := sha256.Sum256([]byte(fmt.Sprintf("%s-%d", expr, index)))
	return fmt.Sprintf("%x", h[:6])[:11]
}

const gridCols = 3

// BuildDashboard creates a Dashboard from a list of panel inputs.
func BuildDashboard(name, scope string, dashID int64, inputs []PanelInput) Dashboard {
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

	return Dashboard{
		ID:    dashID,
		Name:  name,
		Scope: scope,
		Dashboard: DashboardContainer{
			Metadata: DashMetadata{Project: `{"saveVariables":false}`},
			Spec: DashSpec{
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
