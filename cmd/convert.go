package cmd

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/aeltai/odyssey/internal/engine"
	"github.com/spf13/cobra"
)

var convertFlags struct {
	output         string
	name           string
	stsURL         string
	stsToken       string
	metricPrefix   string
	includeMissing bool
	rewriteMetrics bool
	dashID         int64
	interval       string
	variables      []string
}

var convertCmd = &cobra.Command{
	Use:   "convert <dashboard.json> [more.json ...]",
	Short: "Convert Grafana dashboards to STS YAML",
	Long: `Parse one or more Grafana dashboard JSON files, check metric availability
against a live SUSE Observability instance, and generate a ready-to-apply
STS dashboard YAML file.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		output := convertFlags.output
		if output == "" {
			base := strings.TrimSuffix(filepath.Base(args[0]), filepath.Ext(args[0]))
			output = base + ".sts.yaml"
		}
		opts := engine.Options{
			Inputs:            args,
			Output:            output,
			Name:              convertFlags.name,
			STSURL:            convertFlags.stsURL,
			STSToken:          convertFlags.stsToken,
			MetricPrefix:      convertFlags.metricPrefix,
			IncludeMissing:    convertFlags.includeMissing,
			RewriteMetrics:    convertFlags.rewriteMetrics,
			DashID:            convertFlags.dashID,
			Interval:          convertFlags.interval,
			VariableOverrides: parseVariableFlags(convertFlags.variables),
		}
		_, err := engine.Run(opts, os.Stderr)
		if err != nil {
			return fmt.Errorf("conversion failed: %w", err)
		}
		return nil
	},
}

func init() {
	f := convertCmd.Flags()
	f.StringVarP(&convertFlags.output, "output", "o", "", "Output YAML path (default: <input>.sts.yaml)")
	f.StringVar(&convertFlags.name, "name", "", "Dashboard name (default: from Grafana title)")
	f.StringVar(&convertFlags.stsURL, "sts-url", "", "SUSE Observability base URL")
	f.StringVar(&convertFlags.stsToken, "sts-token", "", "SUSE Observability API token")
	f.StringVar(&convertFlags.metricPrefix, "metric-prefix", "", "Metric prefix (auto-detected if empty)")
	f.BoolVar(&convertFlags.includeMissing, "include-missing", false, "Include panels with missing metrics")
	f.BoolVar(&convertFlags.rewriteMetrics, "rewrite-metrics", true, "Rewrite metric names to match STS prefix")
	f.Int64Var(&convertFlags.dashID, "id", 0, "Existing STS dashboard ID (for updates)")
	f.StringVar(&convertFlags.interval, "interval", "5m", "PromQL interval for rate/irate (e.g. 5m, 1m, 15m)")
	f.StringArrayVarP(&convertFlags.variables, "variable", "v", nil, "Bake variable value into queries (e.g. -v namespace=prod -v job=myjob)")
}
