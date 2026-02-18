package engine

import (
	"fmt"
	"io"
	"strings"

	"github.com/aeltai/odyssey/internal/grafana"
	"github.com/aeltai/odyssey/internal/promql"
	"github.com/aeltai/odyssey/internal/sts"
)

// Options controls the conversion behaviour.
type Options struct {
	Inputs            []string
	Output            string
	Name              string
	STSURL            string
	STSToken          string
	MetricPrefix      string
	IncludeMissing    bool
	CheckOnly         bool
	RewriteMetrics    bool
	DashID            int64
	Interval          string            // PromQL interval for rate/irate (e.g. 5m). Default "5m".
	VariableOverrides map[string]string  // User-specified variable overrides (e.g. namespace=prod).
}

// Result holds the outcome of a conversion run.
type Result struct {
	TotalPanels    int
	Matched        int
	Missing        int
	DetectedPrefix string
	OutputPath     string
}

// EnrichedPanel is a parsed panel with sanitised query and extracted metrics.
type EnrichedPanel struct {
	grafana.Panel
	Sanitized   string
	MetricNames []string
	ParseError  bool
}

// ParseInputs reads all Grafana JSON files and returns panels, the first title
// found, and merged variable defaults from templating.list.
func ParseInputs(paths []string) ([]grafana.Panel, string, map[string]string, error) {
	var all []grafana.Panel
	var title string
	varDefaults := map[string]string{}
	for _, p := range paths {
		result, err := grafana.ParseFileResult(p)
		if err != nil {
			return nil, "", nil, err
		}
		if title == "" && result.Title != "" {
			title = result.Title
		}
		for k, v := range result.VariableDefaults {
			if _, exists := varDefaults[k]; !exists {
				varDefaults[k] = v
			}
		}
		all = append(all, result.Panels...)
	}
	if len(varDefaults) == 0 {
		varDefaults = nil
	}
	return all, title, varDefaults, nil
}

// MergeVars merges Grafana variable defaults with user overrides (overrides win).
func MergeVars(defaults, overrides map[string]string) map[string]string {
	if len(defaults) == 0 && len(overrides) == 0 {
		return nil
	}
	merged := make(map[string]string)
	for k, v := range defaults {
		merged[k] = v
	}
	for k, v := range overrides {
		merged[k] = v
	}
	if len(merged) == 0 {
		return nil
	}
	return merged
}

// SanitiseAndExtract sanitises expressions and extracts metric names.
// Warnings are written to w. interval is used for $__interval, $__range_s etc. (default "5m").
// vars holds merged variable values to bake into label selectors (nil = strip all).
func SanitiseAndExtract(panels []grafana.Panel, interval string, vars map[string]string, w io.Writer) []EnrichedPanel {
	if interval == "" {
		interval = "5m"
	}
	out := make([]EnrichedPanel, 0, len(panels))
	for _, p := range panels {
		san := sts.SanitizePromQLWithVars(p.Expr, interval, vars)
		names, err := promql.ExtractMetricNames(san)
		ep := EnrichedPanel{Panel: p, Sanitized: san}
		if err != nil {
			ep.ParseError = true
			fmt.Fprintf(w, "  warn: PromQL parse error in %q: %v\n", p.Title, err)
		} else {
			ep.MetricNames = names
		}
		out = append(out, ep)
	}
	return out
}

// MatchResult holds the per-panel match outcome.
type MatchResult struct {
	Title   string
	Expr    string
	Metrics []string
	HasData bool
	ParseError bool
}

// MatchPanels checks each panel's metrics against the STS index.
func MatchPanels(enriched []EnrichedPanel, idx *sts.MetricIndex) (results []MatchResult, detectedPrefix string) {
	prefixCounts := map[string]int{}

	for _, ep := range enriched {
		mr := MatchResult{
			Title:      ep.Title,
			Expr:       ep.Sanitized,
			Metrics:    ep.MetricNames,
			ParseError: ep.ParseError,
		}
		for _, mn := range ep.MetricNames {
			if ok, actual := idx.Has(mn); ok {
				mr.HasData = true
				if actual != mn {
					prefix := strings.TrimRight(strings.TrimSuffix(actual, mn), "_")
					if prefix != "" {
						prefixCounts[prefix]++
					}
				}
				break
			}
		}
		results = append(results, mr)
	}

	best, bestCount := "", 0
	for p, c := range prefixCounts {
		if c >= 2 && (c > bestCount || (c == bestCount && len(p) > len(best))) {
			best, bestCount = p, c
		}
	}
	return results, best
}

// BuildPanelInputs filters match results into PanelInputs for YAML generation.
func BuildPanelInputs(results []MatchResult, includeMissing bool) []sts.PanelInput {
	var inputs []sts.PanelInput
	for _, mr := range results {
		if mr.HasData || includeMissing {
			inputs = append(inputs, sts.PanelInput{Title: mr.Title, Expr: mr.Expr})
		}
	}
	return inputs
}

// CountResults returns (matched, missing) counts from match results.
func CountResults(results []MatchResult) (int, int) {
	matched, missing := 0, 0
	for _, mr := range results {
		if mr.HasData {
			matched++
		} else {
			missing++
		}
	}
	return matched, missing
}

// Run executes the full conversion pipeline and returns the result.
func Run(opts Options, w io.Writer) (*Result, error) {
	panels, dashTitle, varDefaults, err := ParseInputs(opts.Inputs)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(w, "Parsed %d panels from %d file(s)\n", len(panels), len(opts.Inputs))
	if len(panels) == 0 {
		return &Result{}, nil
	}

	interval := opts.Interval
	if interval == "" {
		interval = "5m"
	}
	vars := MergeVars(varDefaults, opts.VariableOverrides)
	enriched := SanitiseAndExtract(panels, interval, vars, w)

	stsCfg, err := sts.LoadConfig(opts.STSURL, opts.STSToken)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(w, "Fetching metrics from %s ...\n", stsCfg.URL)
	metricIdx, err := sts.FetchAvailableMetrics(stsCfg)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(w, "STS reports %d metric names\n", len(metricIdx.Exact))

	results, detectedPrefix := MatchPanels(enriched, metricIdx)
	matched, missing := CountResults(results)

	if detectedPrefix != "" {
		fmt.Fprintf(w, "Auto-detected metric prefix: %q\n", detectedPrefix)
	}
	fmt.Fprintf(w, "\nResult: %d panels with data, %d missing\n", matched, missing)

	if opts.CheckOnly {
		for _, mr := range results {
			label := "AVAILABLE"
			if !mr.HasData {
				label = "MISSING"
			}
			metrics := strings.Join(mr.Metrics, ", ")
			if mr.ParseError {
				metrics = "(parse error)"
			}
			snippet := mr.Expr
			if len(snippet) > 80 {
				snippet = snippet[:77] + "..."
			}
			fmt.Fprintf(w, "  [%s] %s\n    expr: %s\n    metrics: %s\n", label, mr.Title, snippet, metrics)
		}
		return &Result{TotalPanels: len(panels), Matched: matched, Missing: missing, DetectedPrefix: detectedPrefix}, nil
	}

	panelInputs := BuildPanelInputs(results, opts.IncludeMissing)
	if len(panelInputs) == 0 {
		fmt.Fprintln(w, "No panels to include. Use --include-missing to keep all.")
		return &Result{TotalPanels: len(panels), Matched: matched, Missing: missing, DetectedPrefix: detectedPrefix}, nil
	}

	prefix := opts.MetricPrefix
	if prefix == "" {
		prefix = detectedPrefix
	}
	if opts.RewriteMetrics && prefix != "" {
		fmt.Fprintf(w, "Rewriting queries with prefix %q\n", prefix)
		for i, pi := range panelInputs {
			panelInputs[i].Expr = sts.RewriteMetricPrefix(pi.Expr, prefix, metricIdx)
		}
	}

	name := opts.Name
	if name == "" {
		name = dashTitle
	}
	if name == "" {
		name = "Grafana migrated"
	}

	dash := sts.BuildDashboard(name, "publicDashboard", opts.DashID, panelInputs)
	if err := sts.WriteDashboardYAML(dash, opts.Output); err != nil {
		return nil, fmt.Errorf("write YAML: %w", err)
	}

	fmt.Fprintf(w, "Wrote %s (%d panels)\n", opts.Output, len(panelInputs))
	fmt.Fprintf(w, "Apply: sts dashboard apply --file %s\n", opts.Output)
	return &Result{
		TotalPanels:    len(panels),
		Matched:        matched,
		Missing:        missing,
		DetectedPrefix: detectedPrefix,
		OutputPath:     opts.Output,
	}, nil
}
