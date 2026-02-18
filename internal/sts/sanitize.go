package sts

import (
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"time"
)

// rangeVarPattern matches $variable or ${variable} used inside range vector brackets [...]
// e.g. [$interval], [${interval}], [$__range] — replaces them with 5m
var rangeVarPattern = regexp.MustCompile(`\[\$\{?\w+\}?\]`)

var varLabelFilter = regexp.MustCompile(`,?\s*\w+=~?["'][^"']*\$\w+[^"']*["']|,?\s*\w+=~?\$\w+`)

var danglingComma = regexp.MustCompile(`\{\s*,\s*`)
var trailingComma = regexp.MustCompile(`,\s*\}`)
var emptyBraces = regexp.MustCompile(`\{\s*\}`)

var uppercaseFunc = regexp.MustCompile(`\b(SUM|AVG|MIN|MAX|COUNT|RATE|IRATE|INCREASE|DELTA|ROUND|ABS|CEIL|FLOOR|SQRT|CLAMP|CLAMP_MAX|CLAMP_MIN|TOPK|BOTTOMK|COUNT_VALUES|QUANTILE|STDDEV|STDVAR|ABSENT|LABEL_REPLACE|LABEL_JOIN|SORT|SORT_DESC|CHANGES|RESETS|DERIV|PREDICT_LINEAR|HISTOGRAM_QUANTILE|LAST_OVER_TIME|MAX_OVER_TIME|MIN_OVER_TIME|SUM_OVER_TIME|AVG_OVER_TIME|COUNT_OVER_TIME|STDDEV_OVER_TIME|QUANTILE_OVER_TIME)\b`)

// SanitizePromQL makes a Grafana PromQL expression compatible with SUSE Observability.
// Equivalent to SanitizePromQLWithVars(expr, interval, nil).
func SanitizePromQL(expr string, interval string) string {
	return SanitizePromQLWithVars(expr, interval, nil)
}

// SanitizePromQLWithVars makes a Grafana PromQL expression compatible with SUSE Observability.
//
//   - Replaces $__rate_interval / $__interval with the given interval (default "5m")
//   - Substitutes known variable values into label selectors (e.g. namespace="$namespace" -> namespace="prod")
//   - Removes remaining label selectors that reference unknown Grafana $variables
//   - Lowercases uppercase PromQL function names
//
// vars is a map of variable name -> value. Nil means no substitution (strip all).
func SanitizePromQLWithVars(expr string, interval string, vars map[string]string) string {
	return sanitizePromQL(expr, interval, vars, nil)
}

// SanitizePromQLPreserveVars is like SanitizePromQLWithVars but preserves variable
// references as ${var} for variables listed in preserveSet, instead of baking or stripping.
// This allows STS dashboard variables to be used interactively.
// - preserveSet: variable names to keep as ${name} references
// - overrides: variable names/values to bake in (takes priority over preserveSet)
func SanitizePromQLPreserveVars(expr string, interval string, overrides map[string]string, preserveSet map[string]bool) string {
	return sanitizePromQL(expr, interval, overrides, preserveSet)
}

func sanitizePromQL(expr string, interval string, vars map[string]string, preserveSet map[string]bool) string {
	if expr == "" {
		return expr
	}
	if interval == "" {
		interval = "5m"
	}
	rangeSec := intervalToSeconds(interval)

	// STS natively supports ${__interval} and ${__rate_interval} — normalize to that syntax.
	// Replace $__range_s and $__range which STS does NOT support.
	expr = strings.ReplaceAll(expr, "$__range_s", strconv.Itoa(rangeSec))
	expr = strings.ReplaceAll(expr, "${__range_s}", strconv.Itoa(rangeSec))
	expr = strings.ReplaceAll(expr, "$__range", interval)
	expr = strings.ReplaceAll(expr, "${__range}", interval)

	if preserveSet != nil {
		// Phase 1: Replace preserved variables with unique placeholders to protect them.
		expr = protectPreservedVars(expr, preserveSet)

		// Phase 2: Normalize builtins to STS native syntax via placeholders.
		expr = strings.ReplaceAll(expr, "$__rate_interval", "__PH_BUILTIN_rate_interval__")
		expr = strings.ReplaceAll(expr, "${__rate_interval}", "__PH_BUILTIN_rate_interval__")
		expr = strings.ReplaceAll(expr, "$__interval", "__PH_BUILTIN_interval__")
		expr = strings.ReplaceAll(expr, "${__interval}", "__PH_BUILTIN_interval__")

		// Phase 3: Replace remaining range brackets with interval
		expr = rangeVarPattern.ReplaceAllString(expr, "["+interval+"]")

		// Phase 4: Bake in explicit overrides
		if len(vars) > 0 {
			expr = substituteVariables(expr, vars)
		}

		// Phase 5: Strip remaining unknown variable references
		expr = varLabelFilter.ReplaceAllString(expr, "")
		expr = danglingComma.ReplaceAllString(expr, "{")
		expr = trailingComma.ReplaceAllString(expr, "}")
		expr = emptyBraces.ReplaceAllString(expr, "")

		// Phase 6: Restore placeholders to ${var} syntax
		expr = restorePreservedVars(expr, preserveSet)
		expr = strings.ReplaceAll(expr, "__PH_BUILTIN_rate_interval__", "${__rate_interval}")
		expr = strings.ReplaceAll(expr, "__PH_BUILTIN_interval__", "${__interval}")
	} else {
		// Legacy mode: replace builtins with literals
		expr = strings.ReplaceAll(expr, "$__rate_interval", interval)
		expr = strings.ReplaceAll(expr, "$__interval", interval)
		expr = strings.ReplaceAll(expr, "${__rate_interval}", interval)
		expr = strings.ReplaceAll(expr, "${__interval}", interval)

		expr = rangeVarPattern.ReplaceAllString(expr, "["+interval+"]")

		if len(vars) > 0 {
			expr = substituteVariables(expr, vars)
		}

		expr = varLabelFilter.ReplaceAllString(expr, "")
		expr = danglingComma.ReplaceAllString(expr, "{")
		expr = trailingComma.ReplaceAllString(expr, "}")
		expr = emptyBraces.ReplaceAllString(expr, "")
	}

	expr = uppercaseFunc.ReplaceAllStringFunc(expr, strings.ToLower)
	return expr
}

// protectPreservedVars replaces $var and ${var} references for preserved variables
// with unique placeholders that won't be caught by the variable-stripping regexes.
func protectPreservedVars(expr string, preserve map[string]bool) string {
	for name := range preserve {
		ph := "__PH_VAR_" + name + "__"
		// Handle ${var} syntax first (more specific)
		expr = strings.ReplaceAll(expr, "${"+name+"}", ph)
		// Handle $var syntax (avoid partial matches by checking word boundary)
		re := regexp.MustCompile(`\$` + regexp.QuoteMeta(name) + `\b`)
		expr = re.ReplaceAllString(expr, ph)
	}
	return expr
}

// restorePreservedVars converts placeholders back to ${var} STS syntax.
func restorePreservedVars(expr string, preserve map[string]bool) string {
	for name := range preserve {
		ph := "__PH_VAR_" + name + "__"
		expr = strings.ReplaceAll(expr, ph, "${"+name+"}")
	}
	return expr
}

// substituteVariables replaces $var and ${var} references in label selectors with
// their concrete values. Handles both = and =~ operators.
func substituteVariables(expr string, vars map[string]string) string {
	for name, value := range vars {
		// Match: label="$var", label='$var', label=~"$var", label=~"^$var$"
		// Also handles ${var} syntax.
		pat := fmt.Sprintf(
			`(\w+=~?)["'](\^?)(\$\{?%s\}?)(\$?)["']`,
			regexp.QuoteMeta(name),
		)
		re := regexp.MustCompile(pat)
		expr = re.ReplaceAllStringFunc(expr, func(match string) string {
			parts := re.FindStringSubmatch(match)
			if parts == nil {
				return match
			}
			op := parts[1]     // e.g. "namespace=" or "namespace=~"
			prefix := parts[2] // "^" or ""
			suffix := parts[4] // "$" or ""

			if strings.HasSuffix(op, "=~") {
				// Use the value as-is — it's a regex pattern provided by the user
				if prefix != "" || suffix != "" {
					return op + `"^` + value + `$"`
				}
				return op + `"` + value + `"`
			}
			return op + `"` + value + `"`
		})

		// Also handle bare (unquoted) variable references: label=$var or label=${var}
		barePat := fmt.Sprintf(`(\w+=~?)(?:\$\{%s\}|\$%s)([,}\s]|$)`, regexp.QuoteMeta(name), regexp.QuoteMeta(name))
		bareRe := regexp.MustCompile(barePat)
		expr = bareRe.ReplaceAllString(expr, `${1}"`+value+`"${2}`)
	}
	return expr
}

var regexMetaChars = regexp.MustCompile(`[.*+?^${}()|[\]\\]`)

func escapeRegex(s string) string {
	return regexMetaChars.ReplaceAllString(s, `\$0`)
}

// intervalToSeconds parses Prometheus-style duration (e.g. 5m, 1h) and returns seconds.
// Falls back to 300 (5m) on parse error.
func intervalToSeconds(interval string) int {
	d, err := time.ParseDuration(interval)
	if err != nil {
		return 300
	}
	return int(d.Seconds())
}

// RewriteMetricPrefix replaces bare metric names in a PromQL expression with
// their prefixed counterparts found in STS (e.g. pg_up -> postgresql_pg_up).
// Also handles _total suffix stripping done by the openmetrics agent.
func RewriteMetricPrefix(expr string, prefix string, idx *MetricIndex) string {
	if expr == "" || prefix == "" || idx == nil {
		return expr
	}
	names := extractMetricNamesFromExpr(expr)
	for _, name := range names {
		if idx.Exact[name] {
			continue
		}
		// Try prefix + name
		prefixed := prefix + "_" + name
		if idx.Exact[prefixed] {
			expr = replaceMetricName(expr, name, prefixed)
			continue
		}
		// Try prefix + name without _total (openmetrics strips counter suffix)
		if strings.HasSuffix(name, "_total") {
			bare := strings.TrimSuffix(name, "_total")
			prefixed = prefix + "_" + bare
			if idx.Exact[prefixed] {
				expr = replaceMetricName(expr, name, prefixed)
				continue
			}
			if idx.Exact[bare] {
				expr = replaceMetricName(expr, name, bare)
			}
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
