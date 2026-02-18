package promql

import (
	"github.com/prometheus/prometheus/promql/parser"
)

// ExtractMetricNames parses a PromQL expression and returns all metric names it references.
func ExtractMetricNames(expr string) ([]string, error) {
	parsed, err := parser.ParseExpr(expr)
	if err != nil {
		return nil, err
	}
	seen := map[string]bool{}
	var names []string
	parser.Inspect(parsed, func(node parser.Node, path []parser.Node) error {
		if n, ok := node.(*parser.VectorSelector); ok {
			if n.Name != "" && !seen[n.Name] {
				seen[n.Name] = true
				names = append(names, n.Name)
			}
		}
		return nil
	})
	return names, nil
}
