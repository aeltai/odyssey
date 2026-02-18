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
	Inputs         []string
	Output         string
	Name           string
	STSURL         string
	STSToken       string
	MetricPrefix   string
	IncludeMissing bool
	CheckOnly      bool
	RewriteMetrics bool
	DashID         int64
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

// ParseInputs reads all Grafana JSON files and returns panels + the first title found.
func ParseInputs(paths []string) ([]grafana.Panel, string, error) {
	var all []grafana.Panel
	var title string
	for _, p := range paths {
		t, panels, err := grafana.ParseFile(p)
		if err != nil {
			return nil, "", err
		}
		if title == "" && t != "" {
			title = t
		}
		all = append(all, panels...)
	}
	return all, title, nil
}

// SanitiseAndExtract sanitises expressions and extracts metric names.
// Warnings are written to w.
func SanitiseAndExtract(panels []grafana.Panel, w io.Writer) []EnrichedPanel {
	out := make([]EnrichedPanel, 0, len(panels))
	for _, p := range panels {
		san := sts.SanitizePromQL(p.Expr)
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
	panels, dashTitle, err := ParseInputs(opts.Inputs)
	if err != nil {
		return nil, err
	}
	fmt.Fprintf(w, "Parsed %d panels from %d file(s)\n", len(panels), len(opts.Inputs))
	if len(panels) == 0 {
		return &Result{}, nil
	}

	enriched := SanitiseAndExtract(panels, w)

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
