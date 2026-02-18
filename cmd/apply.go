package cmd

import (
	"github.com/spf13/cobra"
)

var applyCmd = &cobra.Command{
	Use:   "apply <dashboard.sts.yaml>",
	Short: "Apply a dashboard YAML to SUSE Observability via sts CLI",
	Long: `Apply a generated STS dashboard YAML file using the sts CLI.
Requires the sts CLI to be installed and configured:

  sts context save --name default --url <URL> --api-token <TOKEN>

This is a convenience wrapper around: sts dashboard apply --file <file>`,
	Args: cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		return applyDashboard(args[0])
	},
}
