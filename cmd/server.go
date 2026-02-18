package cmd

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/aeltai/odyssey/internal/api"
	"github.com/spf13/cobra"
)

var serverPort int

//go:embed all:dist
var distFS embed.FS

var serverCmd = &cobra.Command{
	Use:   "server",
	Short: "Start the web UI",
	Long:  "Start a local web server with the Odyssey dashboard conversion UI.",
	RunE: func(cmd *cobra.Command, args []string) error {
		router := api.NewRouter()

		sub, err := fs.Sub(distFS, "dist")
		if err != nil {
			return fmt.Errorf("embedded frontend: %w", err)
		}
		frontend := api.ServeEmbedded(http.FS(sub))

		mux := http.NewServeMux()
		mux.Handle("/api/", router)
		mux.Handle("/", frontend)

		addr := fmt.Sprintf(":%d", serverPort)
		fmt.Printf("Odyssey web UI → http://localhost%s\n", addr)
		return http.ListenAndServe(addr, mux)
	},
}

func init() {
	serverCmd.Flags().IntVarP(&serverPort, "port", "p", 3000, "Port to listen on")
	rootCmd.AddCommand(serverCmd)
}
