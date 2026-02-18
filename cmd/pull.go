package cmd

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/spf13/cobra"
)

var pullFlags struct {
	grafanaURL   string
	grafanaToken string
	outputDir    string
	uid          string
}

var pullCmd = &cobra.Command{
	Use:   "pull [--uid <dashboard-uid>]",
	Short: "Pull dashboards from a Grafana instance",
	Long: `Connect to a Grafana instance and download dashboard JSON files.
Without --uid, lists all available dashboards. With --uid, fetches a
specific dashboard and saves it as a JSON file ready for 'odyssey convert'.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		if pullFlags.grafanaURL == "" {
			return fmt.Errorf("--grafana-url is required")
		}
		url := strings.TrimRight(pullFlags.grafanaURL, "/")

		if pullFlags.uid != "" {
			return pullSingleDashboard(url, pullFlags.grafanaToken, pullFlags.uid, pullFlags.outputDir)
		}
		return listGrafanaDashboards(url, pullFlags.grafanaToken)
	},
}

func init() {
	f := pullCmd.Flags()
	f.StringVar(&pullFlags.grafanaURL, "grafana-url", "", "Grafana base URL (e.g. https://grafana.example.com)")
	f.StringVar(&pullFlags.grafanaToken, "grafana-token", "", "Grafana API key or service account token")
	f.StringVar(&pullFlags.uid, "uid", "", "Dashboard UID to fetch (omit to list all)")
	f.StringVarP(&pullFlags.outputDir, "output-dir", "o", ".", "Directory to save downloaded JSON files")
}

func listGrafanaDashboards(baseURL, token string) error {
	apiURL := baseURL + "/api/search?type=dash-db&limit=200"

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Grafana returned %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var dashboards []struct {
		UID   string   `json:"uid"`
		Title string   `json:"title"`
		URI   string   `json:"uri"`
		Tags  []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dashboards); err != nil {
		return fmt.Errorf("failed to parse response: %w", err)
	}

	fmt.Fprintf(os.Stderr, "Found %d dashboards on %s\n\n", len(dashboards), baseURL)
	for _, d := range dashboards {
		tags := ""
		if len(d.Tags) > 0 {
			tags = dimStyle.Render(" [" + strings.Join(d.Tags, ", ") + "]")
		}
		fmt.Printf("  %-20s %s%s\n", d.UID, d.Title, tags)
	}

	fmt.Fprintf(os.Stderr, "\nTo download a dashboard:\n  odyssey pull --grafana-url %s --uid <UID>\n", baseURL)
	return nil
}

func pullSingleDashboard(baseURL, token, uid, outputDir string) error {
	apiURL := baseURL + "/api/dashboards/uid/" + uid

	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return err
	}
	req.Header.Set("Accept", "application/json")
	if token != "" {
		req.Header.Set("Authorization", "Bearer "+token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("Grafana returned %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var wrapper struct {
		Dashboard json.RawMessage `json:"dashboard"`
		Meta      struct {
			Slug string `json:"slug"`
		} `json:"meta"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return fmt.Errorf("failed to parse dashboard: %w", err)
	}

	slug := wrapper.Meta.Slug
	if slug == "" {
		slug = uid
	}
	filename := filepath.Join(outputDir, slug+".json")

	// Pretty-print the JSON
	var pretty json.RawMessage
	if err := json.Unmarshal(wrapper.Dashboard, &pretty); err == nil {
		indented, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			wrapper.Dashboard = indented
		}
	}

	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		return err
	}
	if err := os.WriteFile(filename, wrapper.Dashboard, 0o644); err != nil {
		return err
	}

	fmt.Fprintf(os.Stderr, "Downloaded %s → %s\n", uid, filename)
	fmt.Fprintf(os.Stderr, "Convert with:\n  odyssey convert %s\n", filename)
	return nil
}

func truncate(s string, n int) string {
	if len(s) <= n {
		return s
	}
	return s[:n] + "..."
}
