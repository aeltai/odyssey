package sts

import (
	"regexp"
	"strings"
)

var varLabelFilter = regexp.MustCompile(`,?\s*\w+=~?["'][^"']*\$\w+[^"']*["']|,?\s*\w+=~?\$\w+`)

var danglingComma = regexp.MustCompile(`\{\s*,\s*`)
var trailingComma = regexp.MustCompile(`,\s*\}`)
var emptyBraces = regexp.MustCompile(`\{\s*\}`)

var uppercaseFunc = regexp.MustCompile(`\b(SUM|AVG|MIN|MAX|COUNT|RATE|IRATE|INCREASE|DELTA|ROUND|ABS|CEIL|FLOOR|SQRT|CLAMP|CLAMP_MAX|CLAMP_MIN|TOPK|BOTTOMK|COUNT_VALUES|QUANTILE|STDDEV|STDVAR|ABSENT|LABEL_REPLACE|LABEL_JOIN|SORT|SORT_DESC|CHANGES|RESETS|DERIV|PREDICT_LINEAR|HISTOGRAM_QUANTILE|LAST_OVER_TIME|MAX_OVER_TIME|MIN_OVER_TIME|SUM_OVER_TIME|AVG_OVER_TIME|COUNT_OVER_TIME|STDDEV_OVER_TIME|QUANTILE_OVER_TIME)\b`)

// SanitizePromQL makes a Grafana PromQL expression compatible with SUSE Observability.
//
//   - Replaces $__rate_interval / $__interval with 5m
//   - Removes label selectors that reference Grafana $variables
//   - Lowercases uppercase PromQL function names
func SanitizePromQL(expr string) string {
	if expr == "" {
		return expr
	}
	expr = strings.ReplaceAll(expr, "$__rate_interval", "5m")
	expr = strings.ReplaceAll(expr, "$__interval", "5m")
	expr = strings.ReplaceAll(expr, "${__rate_interval}", "5m")
	expr = strings.ReplaceAll(expr, "${__interval}", "5m")

	expr = varLabelFilter.ReplaceAllString(expr, "")
	expr = danglingComma.ReplaceAllString(expr, "{")
	expr = trailingComma.ReplaceAllString(expr, "}")
	expr = emptyBraces.ReplaceAllString(expr, "")

	expr = uppercaseFunc.ReplaceAllStringFunc(expr, strings.ToLower)
	return expr
}

// RewriteMetricPrefix replaces bare metric names in a PromQL expression with
// their prefixed counterparts found in STS (e.g. pg_up -> postgresql_pg_up).
func RewriteMetricPrefix(expr string, prefix string, idx *MetricIndex) string {
	if expr == "" || prefix == "" || idx == nil {
		return expr
	}
	names := extractMetricNamesFromExpr(expr)
	for _, name := range names {
		if idx.Exact[name] {
			continue
		}
		prefixed := prefix + "_" + name
		if idx.Exact[prefixed] {
			expr = replaceMetricName(expr, name, prefixed)
		}
	}
	return expr
}

func extractMetricNamesFromExpr(expr string) []string {
	re := regexp.MustCompile(`\b([a-zA-Z_:][a-zA-Z0-9_:]*)\s*[\{(\[]?`)
	matches := re.FindAllStringSubmatch(expr, -1)
	seen := map[string]bool{}
	promqlKeywords := map[string]bool{
		"sum": true, "avg": true, "min": true, "max": true, "count": true,
		"rate": true, "irate": true, "increase": true, "delta": true,
		"histogram_quantile": true, "topk": true, "bottomk": true,
		"count_values": true, "quantile": true, "stddev": true, "stdvar": true,
		"absent": true, "absent_over_time": true, "ceil": true, "floor": true,
		"round": true, "clamp": true, "clamp_max": true, "clamp_min": true,
		"label_replace": true, "label_join": true, "sort": true, "sort_desc": true,
		"time": true, "timestamp": true, "vector": true, "scalar": true,
		"sgn": true, "sqrt": true, "exp": true, "ln": true, "log2": true, "log10": true,
		"abs": true, "changes": true, "deriv": true, "predict_linear": true,
		"resets": true, "idelta": true, "last_over_time": true, "present_over_time": true,
		"max_over_time": true, "min_over_time": true, "sum_over_time": true,
		"avg_over_time": true, "count_over_time": true, "stddev_over_time": true,
		"stdvar_over_time": true, "quantile_over_time": true,
		"by": true, "without": true, "on": true, "ignoring": true, "group_left": true,
		"group_right": true, "bool": true, "and": true, "or": true, "unless": true,
		"offset": true, "SUM": true, "AVG": true, "MIN": true, "MAX": true, "COUNT": true,
	}
	var names []string
	for _, m := range matches {
		name := m[1]
		if !seen[name] && !promqlKeywords[name] && !promqlKeywords[strings.ToLower(name)] {
			seen[name] = true
			names = append(names, name)
		}
	}
	return names
}

func replaceMetricName(expr, old, replacement string) string {
	re := regexp.MustCompile(`\b` + regexp.QuoteMeta(old) + `\b`)
	return re.ReplaceAllString(expr, replacement)
}
