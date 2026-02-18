package cmd

import (
	"fmt"
	"os"

	"github.com/spf13/cobra"
)

var version = "dev"

func SetVersion(v string) { version = v }

var rootCmd = &cobra.Command{
	Use:   "odyssey",
	Short: "Convert Grafana dashboards to SUSE Observability YAML",
	Long: `Odyssey converts Grafana dashboard JSON files into SUSE Observability
(StackState) dashboard YAML. It parses PromQL expressions, checks metric
availability against a live STS instance, sanitises queries, and generates
ready-to-apply YAML.

Run without a subcommand to enter the interactive wizard.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		return runInteractive()
	},
}

func Execute() {
	if err := rootCmd.Execute(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func init() {
	rootCmd.AddCommand(convertCmd)
	rootCmd.AddCommand(checkCmd)
	rootCmd.AddCommand(pullCmd)
	rootCmd.AddCommand(applyCmd)
	rootCmd.AddCommand(versionCmd)
}

var versionCmd = &cobra.Command{
	Use:   "version",
	Short: "Print version",
	Run: func(cmd *cobra.Command, args []string) {
		fmt.Println("odyssey", version)
	},
}
