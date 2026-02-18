package sts

import "testing"

func TestSanitizePromQL(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		expect string
	}{
		{"rate_interval replaced", `rate(x[$__rate_interval])`, `rate(x[5m])`},
		{"interval with braces", `rate(x{a="b"}[${__interval}])`, `rate(x{a="b"}[5m])`},
		{"both interval variants", `rate(a[$__rate_interval]) + rate(b[${__interval}])`, `rate(a[5m]) + rate(b[5m])`},
		{"variable labels removed", `node_cpu{instance="$node",job="$job"}`, `node_cpu`},
		{"single-quoted variable labels", `node_cpu{instance=~'$node:$port',job=~'$job'}`, `node_cpu`},
		{"kubernetes variable pattern", `container_cpu{kubernetes_io_hostname=~"^$Node$"}`, `container_cpu`},
		{"mixed real and variable labels", `pg_stat{datname=~".*",release=~"$release",instance=~"$instance"}`, `pg_stat{datname=~".*"}`},
		{"keep non-variable labels", `pg_stat{state="active",instance=~"$instance"}`, `pg_stat{state="active"}`},
		{"uppercase SUM lowered", `SUM(rate(x[5m]))`, `sum(rate(x[5m]))`},
		{"uppercase AVG lowered", `AVG(x) by (host)`, `avg(x) by (host)`},
		{"uppercase HISTOGRAM_QUANTILE", `HISTOGRAM_QUANTILE(0.99, SUM(RATE(h[5m])) by (le))`, `histogram_quantile(0.99, sum(rate(h[5m])) by (le))`},
		{"already lowercase unchanged", `sum(rate(x[5m]))`, `sum(rate(x[5m]))`},
		{"no labels no change", `up`, `up`},
		{"empty expr", "", ""},
		{"variable without quotes", `foo{bar=$baz}`, `foo`},
		{"over_time functions lowered", `MAX_OVER_TIME(x[1h]) / MIN_OVER_TIME(y[1h])`, `max_over_time(x[1h]) / min_over_time(y[1h])`},
		{"label at start of selector", `metric{release=~"$release",mode="idle"}`, `metric{mode="idle"}`},
		{"bare $interval in range", `rate(x{instance="$host"}[$interval])`, `rate(x[5m])`},
		{"bare ${interval} in range", `rate(x[${interval}])`, `rate(x[5m])`},
		{"$interval with or fallback", `rate(x[$interval]) or irate(x[5m])`, `rate(x[5m]) or irate(x[5m])`},
		{"$__range variable", `max_over_time(x[$__range])`, `max_over_time(x[5m])`},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := SanitizePromQL(tt.input)
			if got != tt.expect {
				t.Errorf("SanitizePromQL(%q)\n  got  %q\n  want %q", tt.input, got, tt.expect)
			}
		})
	}
}

func TestRewriteMetricPrefix(t *testing.T) {
	idx := &MetricIndex{
		Exact: map[string]bool{
			"postgresql_pg_up":                     true,
			"postgresql_pg_stat_user":              true,
			"up":                                   true,
			"mysql_mysql_global_status_commands":    true,
		},
		Suffix: map[string]string{
			"pg_up":                       "postgresql_pg_up",
			"pg_stat_user":                "postgresql_pg_stat_user",
			"mysql_global_status_commands": "mysql_mysql_global_status_commands",
		},
	}

	tests := []struct {
		name, expr, prefix, expect string
		nilIdx                     bool
	}{
		{"rewrites bare metric", `rate(pg_up[5m])`, "postgresql", `rate(postgresql_pg_up[5m])`, false},
		{"no rewrite for existing metric", `up`, "postgresql", `up`, false},
		{"empty expression", "", "postgresql", "", false},
		{"empty prefix", `pg_up`, "", `pg_up`, false},
		{"nil index", `pg_up`, "postgresql", `pg_up`, true},
		{"rewrites _total to stripped counter", `rate(mysql_global_status_commands_total[5m])`, "mysql", `rate(mysql_mysql_global_status_commands[5m])`, false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var testIdx *MetricIndex
			if !tt.nilIdx {
				testIdx = idx
			}
			got := RewriteMetricPrefix(tt.expr, tt.prefix, testIdx)
			if got != tt.expect {
				t.Errorf("got %q, want %q", got, tt.expect)
			}
		})
	}
}
