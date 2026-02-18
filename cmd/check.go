package cmd

import (
	"fmt"
	"os"

	"github.com/aeltai/odyssey/internal/engine"
	"github.com/spf13/cobra"
)

var checkFlags struct {
	stsURL    string
	stsToken  string
	interval  string
	variables []string
}

var checkCmd = &cobra.Command{
	Use:   "check <dashboard.json> [more.json ...]",
	Short: "Check metric availability without generating YAML",
	Long: `Parse Grafana dashboard JSON files and report which panels have metrics
available in the connected SUSE Observability instance. No YAML is generated.`,
	Args: cobra.MinimumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		opts := engine.Options{
			Inputs:            args,
			CheckOnly:         true,
			STSURL:            checkFlags.stsURL,
			STSToken:          checkFlags.stsToken,
			Interval:          checkFlags.interval,
			VariableOverrides: parseVariableFlags(checkFlags.variables),
		}
		result, err := engine.Run(opts, os.Stderr)
		if err != nil {
			return fmt.Errorf("check failed: %w", err)
		}
		if result != nil {
			fmt.Fprintf(os.Stderr, "\nSummary: %d/%d panels have data\n", result.Matched, result.TotalPanels)
		}
		return nil
	},
}

func init() {
	f := checkCmd.Flags()
	f.StringVar(&checkFlags.stsURL, "sts-url", "", "SUSE Observability base URL")
	f.StringVar(&checkFlags.stsToken, "sts-token", "", "SUSE Observability API token")
	f.StringVar(&checkFlags.interval, "interval", "5m", "PromQL interval for rate/irate (e.g. 5m, 1m, 15m)")
	f.StringArrayVarP(&checkFlags.variables, "variable", "v", nil, "Bake variable value into queries (e.g. -v namespace=prod)")
}
