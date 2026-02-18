package promql

import (
	"sort"
	"testing"
)

func TestExtractMetricNames(t *testing.T) {
	tests := []struct {
		name   string
		expr   string
		expect []string
		errOK  bool
	}{
		{
			name:   "simple selector",
			expr:   `node_cpu_seconds_total{mode="idle"}`,
			expect: []string{"node_cpu_seconds_total"},
		},
		{
			name:   "rate with matrix",
			expr:   `rate(node_network_receive_bytes_total[5m])`,
			expect: []string{"node_network_receive_bytes_total"},
		},
		{
			name:   "multiple metrics",
			expr:   `node_memory_MemTotal_bytes - node_memory_MemFree_bytes`,
			expect: []string{"node_memory_MemFree_bytes", "node_memory_MemTotal_bytes"},
		},
		{
			name:   "nested functions",
			expr:   `sum(rate(container_cpu_usage_seconds_total{id="/"}[5m])) / sum(machine_cpu_cores)`,
			expect: []string{"container_cpu_usage_seconds_total", "machine_cpu_cores"},
		},
		{
			name:  "invalid expr",
			expr:  `not valid promql {{{`,
			errOK: true,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := ExtractMetricNames(tt.expr)
			if tt.errOK {
				if err == nil {
					t.Error("expected error, got nil")
				}
				return
			}
			if err != nil {
				t.Fatalf("unexpected error: %v", err)
			}
			sort.Strings(got)
			sort.Strings(tt.expect)
			if len(got) != len(tt.expect) {
				t.Fatalf("got %v, want %v", got, tt.expect)
			}
			for i := range got {
				if got[i] != tt.expect[i] {
					t.Errorf("got[%d]=%s, want %s", i, got[i], tt.expect[i])
				}
			}
		})
	}
}
