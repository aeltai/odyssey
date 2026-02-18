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

	"github.com/aeltai/odyssey/internal/engine"
	"github.com/aeltai/odyssey/internal/grafana"
	"github.com/aeltai/odyssey/internal/sts"
	"github.com/charmbracelet/huh"
	"github.com/charmbracelet/lipgloss"
)

var (
	titleStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#7D56F4")).
			MarginBottom(1)

	successStyle = lipgloss.NewStyle().
			Bold(true).
			Foreground(lipgloss.Color("#04B575"))

	warnStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#FF8C00"))

	dimStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#666666"))
)

func runInteractive() error {
	fmt.Println(titleStyle.Render("⛵ Odyssey — Grafana → SUSE Observability Dashboard Converter"))
	fmt.Println(dimStyle.Render("Interactive mode. Use 'odyssey convert' for non-interactive usage.\n"))

	// --- Step 1: Choose input source ---
	var inputSource string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Where is your Grafana dashboard?").
				Options(
					huh.NewOption("Local JSON file", "file"),
					huh.NewOption("Pull from Grafana instance", "grafana"),
				).
				Value(&inputSource),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	var panels []grafana.Panel
	var dashTitle string
	var inputs []string

	if inputSource == "grafana" {
		p, t, ins, err := interactiveGrafanaPull()
		if err != nil {
			return err
		}
		panels, dashTitle, inputs = p, t, ins
	} else {
		p, t, ins, err := interactiveFileParse()
		if err != nil {
			return err
		}
		panels, dashTitle, inputs = p, t, ins
	}

	if len(panels) == 0 {
		fmt.Println(warnStyle.Render("No panels with PromQL expressions found."))
		return nil
	}

	// --- Step 3: STS connection ---
	stsURL, stsToken := "", ""
	existingCfg, cfgErr := sts.LoadConfig("", "")
	hasExisting := cfgErr == nil && existingCfg.URL != ""

	if hasExisting {
		var useExisting bool
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewConfirm().
					Title(fmt.Sprintf("Use detected STS connection? (%s)", existingCfg.URL)).
					Value(&useExisting),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}
		if useExisting {
			stsURL = existingCfg.URL
			stsToken = existingCfg.Token
		}
	}

	if stsURL == "" {
		err = huh.NewForm(
			huh.NewGroup(
				huh.NewInput().
					Title("SUSE Observability URL").
					Placeholder("https://observability.example.com").
					Value(&stsURL).
					Validate(func(s string) error {
						if !strings.HasPrefix(s, "http") {
							return fmt.Errorf("URL must start with http:// or https://")
						}
						return nil
					}),
				huh.NewInput().
					Title("API Token").
					Placeholder("your-api-token").
					EchoMode(huh.EchoModePassword).
					Value(&stsToken).
					Validate(func(s string) error {
						if strings.TrimSpace(s) == "" {
							return fmt.Errorf("token is required")
						}
						return nil
					}),
			),
		).WithTheme(huh.ThemeCatppuccin()).Run()
		if err != nil {
			return err
		}
	}

	// --- Step 4: Fetch metrics and show report ---
	stsCfg, err := sts.LoadConfig(stsURL, stsToken)
	if err != nil {
		return err
	}
	fmt.Printf("  Connecting to %s ...\n", dimStyle.Render(stsCfg.URL))
	metricIdx, err := sts.FetchAvailableMetrics(stsCfg)
	if err != nil {
		return fmt.Errorf("failed to fetch metrics: %w", err)
	}
	fmt.Printf("  STS reports %s\n\n", successStyle.Render(fmt.Sprintf("%d metrics", len(metricIdx.Exact))))

	enriched := engine.SanitiseAndExtract(panels, "5m", nil, os.Stderr)
	results, detectedPrefix := engine.MatchPanels(enriched, metricIdx)
	matched, missing := engine.CountResults(results)

	if detectedPrefix != "" {
		fmt.Printf("  Auto-detected metric prefix: %s\n", successStyle.Render(detectedPrefix))
	}
	fmt.Printf("  Panels with data: %s  |  Missing: %s\n\n",
		successStyle.Render(fmt.Sprintf("%d", matched)),
		warnStyle.Render(fmt.Sprintf("%d", missing)))

	limit := len(results)
	if limit > 10 {
		limit = 10
	}
	for _, mr := range results[:limit] {
		icon := successStyle.Render("✓")
		if !mr.HasData {
			icon = warnStyle.Render("✗")
		}
		fmt.Printf("  %s %s\n", icon, mr.Title)
	}
	if len(results) > 10 {
		fmt.Printf("  %s\n", dimStyle.Render(fmt.Sprintf("  ... and %d more panels", len(results)-10)))
	}
	fmt.Println()

	// --- Step 5: Configuration ---
	var (
		dashName       string
		includeMissing bool
		rewriteMetrics bool
		outputPath     string
	)

	defaultName := dashTitle
	if defaultName == "" {
		defaultName = "Grafana migrated"
	}
	dashName = defaultName

	defaultOutput := strings.TrimSuffix(filepath.Base(inputs[0]), filepath.Ext(inputs[0])) + ".sts.yaml"
	outputPath = defaultOutput

	err = huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Dashboard name in STS").
				Value(&dashName),
			huh.NewInput().
				Title("Output YAML file").
				Value(&outputPath),
			huh.NewConfirm().
				Title("Include panels with missing metrics?").
				Description("If yes, panels without data in STS are still included").
				Value(&includeMissing),
			huh.NewConfirm().
				Title("Rewrite metric names with detected prefix?").
				Description(fmt.Sprintf("Prefix: %q", detectedPrefix)).
				Affirmative("Yes").
				Negative("No").
				Value(&rewriteMetrics),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	// --- Step 6: Generate ---
	panelInputs := engine.BuildPanelInputs(results, includeMissing)
	if len(panelInputs) == 0 {
		fmt.Println(warnStyle.Render("No panels to include."))
		return nil
	}

	prefix := detectedPrefix
	if rewriteMetrics && prefix != "" {
		for i, pi := range panelInputs {
			panelInputs[i].Expr = sts.RewriteMetricPrefix(pi.Expr, prefix, metricIdx)
		}
	}

	dash := sts.BuildDashboard(dashName, "publicDashboard", 0, panelInputs)
	if err := sts.WriteDashboardYAML(dash, outputPath); err != nil {
		return fmt.Errorf("write YAML: %w", err)
	}

	fmt.Println()
	fmt.Println(successStyle.Render(fmt.Sprintf("✓ Wrote %s with %d panels", outputPath, len(panelInputs))))
	fmt.Println()

	// --- Step 7: Apply with sts CLI ---
	var applyNow bool
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewConfirm().
				Title("Apply dashboard to STS now?").
				Description("Runs: sts dashboard apply --file " + outputPath).
				Value(&applyNow),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return err
	}

	if applyNow {
		return applyDashboard(outputPath)
	}

	fmt.Println(dimStyle.Render("Apply later with:"))
	fmt.Printf("  sts dashboard apply --file %s\n", outputPath)
	fmt.Printf("  odyssey apply %s\n\n", outputPath)
	return nil
}

func interactiveFileParse() ([]grafana.Panel, string, []string, error) {
	var inputPath string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Grafana dashboard JSON file").
				Description("Path to a Grafana .json export (supports globs like *.json)").
				Placeholder("./dashboard.json").
				Value(&inputPath).
				Validate(func(s string) error {
					if strings.TrimSpace(s) == "" {
						return fmt.Errorf("path is required")
					}
					matches, _ := filepath.Glob(s)
					if len(matches) == 0 {
						if _, err := os.Stat(s); err != nil {
							return fmt.Errorf("file not found: %s", s)
						}
					}
					return nil
				}),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return nil, "", nil, err
	}

	inputs, _ := filepath.Glob(inputPath)
	if len(inputs) == 0 {
		inputs = []string{inputPath}
	}

	parsed, err := engine.ParseInputs(inputs)
	if err != nil {
		return nil, "", nil, fmt.Errorf("parse error: %w", err)
	}
	fmt.Printf("  Found %s in %d file(s)\n\n",
		successStyle.Render(fmt.Sprintf("%d panels", len(parsed.Panels))),
		len(inputs))

	return parsed.Panels, parsed.Title, inputs, nil
}

func interactiveGrafanaPull() ([]grafana.Panel, string, []string, error) {
	var grafanaURL, grafanaToken string
	err := huh.NewForm(
		huh.NewGroup(
			huh.NewInput().
				Title("Grafana URL").
				Placeholder("https://grafana.example.com").
				Value(&grafanaURL).
				Validate(func(s string) error {
					if !strings.HasPrefix(s, "http") {
						return fmt.Errorf("URL must start with http:// or https://")
					}
					return nil
				}),
			huh.NewInput().
				Title("Grafana API Token (optional for public)").
				Placeholder("glsa_...").
				EchoMode(huh.EchoModePassword).
				Value(&grafanaToken),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return nil, "", nil, err
	}

	grafanaURL = strings.TrimRight(grafanaURL, "/")
	fmt.Printf("  Connecting to %s ...\n", dimStyle.Render(grafanaURL))

	// List dashboards
	apiURL := grafanaURL + "/api/search?type=dash-db&limit=200"
	req, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		return nil, "", nil, err
	}
	req.Header.Set("Accept", "application/json")
	if grafanaToken != "" {
		req.Header.Set("Authorization", "Bearer "+grafanaToken)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return nil, "", nil, fmt.Errorf("Grafana connection failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, "", nil, fmt.Errorf("Grafana returned %d: %s", resp.StatusCode, truncate(string(body), 300))
	}

	var dashList []struct {
		UID   string `json:"uid"`
		Title string `json:"title"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&dashList); err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse response: %w", err)
	}

	if len(dashList) == 0 {
		return nil, "", nil, fmt.Errorf("no dashboards found on this Grafana instance")
	}

	fmt.Printf("  Found %s\n\n", successStyle.Render(fmt.Sprintf("%d dashboards", len(dashList))))

	// Build options for selection
	options := make([]huh.Option[string], len(dashList))
	for i, d := range dashList {
		options[i] = huh.NewOption(d.Title, d.UID)
	}

	var selectedUID string
	err = huh.NewForm(
		huh.NewGroup(
			huh.NewSelect[string]().
				Title("Select a dashboard to migrate").
				Options(options...).
				Value(&selectedUID),
		),
	).WithTheme(huh.ThemeCatppuccin()).Run()
	if err != nil {
		return nil, "", nil, err
	}

	// Find the title
	var selectedTitle string
	for _, d := range dashList {
		if d.UID == selectedUID {
			selectedTitle = d.Title
			break
		}
	}

	fmt.Printf("  Fetching %s ...\n", dimStyle.Render(selectedTitle))

	// Fetch dashboard
	fetchURL := grafanaURL + "/api/dashboards/uid/" + selectedUID
	fetchReq, err := http.NewRequest("GET", fetchURL, nil)
	if err != nil {
		return nil, "", nil, err
	}
	fetchReq.Header.Set("Accept", "application/json")
	if grafanaToken != "" {
		fetchReq.Header.Set("Authorization", "Bearer "+grafanaToken)
	}

	fetchResp, err := client.Do(fetchReq)
	if err != nil {
		return nil, "", nil, fmt.Errorf("fetch failed: %w", err)
	}
	defer fetchResp.Body.Close()

	if fetchResp.StatusCode != 200 {
		body, _ := io.ReadAll(fetchResp.Body)
		return nil, "", nil, fmt.Errorf("Grafana returned %d: %s", fetchResp.StatusCode, truncate(string(body), 300))
	}

	var wrapper struct {
		Dashboard json.RawMessage `json:"dashboard"`
		Meta      struct {
			Slug string `json:"slug"`
		} `json:"meta"`
	}
	body, _ := io.ReadAll(fetchResp.Body)
	if err := json.Unmarshal(body, &wrapper); err != nil {
		return nil, "", nil, fmt.Errorf("failed to parse dashboard: %w", err)
	}

	// Save to local file
	slug := wrapper.Meta.Slug
	if slug == "" {
		slug = selectedUID
	}
	localFile := slug + ".json"

	var pretty json.RawMessage
	if err := json.Unmarshal(wrapper.Dashboard, &pretty); err == nil {
		indented, err := json.MarshalIndent(pretty, "", "  ")
		if err == nil {
			wrapper.Dashboard = indented
		}
	}
	os.WriteFile(localFile, wrapper.Dashboard, 0o644)
	fmt.Printf("  Saved to %s\n", dimStyle.Render(localFile))

	// Parse
	title, panels, err := grafana.ParseBytes(wrapper.Dashboard)
	if err != nil {
		return nil, "", nil, fmt.Errorf("parse error: %w", err)
	}

	fmt.Printf("  Found %s\n\n", successStyle.Render(fmt.Sprintf("%d panels", len(panels))))

	return panels, title, []string{localFile}, nil
}

func applyDashboard(path string) error {
	fmt.Printf("  Applying %s ...\n", path)

	cmd := findSTSBinary()
	if cmd == "" {
		return fmt.Errorf("sts CLI not found in PATH or current directory. Install it first:\n  curl -sSL https://dl.stackstate.com/sts-cli/install.sh | bash")
	}

	proc := newCommand(cmd, "dashboard", "apply", "--file", path)
	proc.Stdout = os.Stdout
	proc.Stderr = os.Stderr
	if err := proc.Run(); err != nil {
		return fmt.Errorf("sts dashboard apply failed: %w", err)
	}
	return nil
}

func findSTSBinary() string {
	if _, err := os.Stat("./sts"); err == nil {
		return "./sts"
	}
	for _, dir := range filepath.SplitList(os.Getenv("PATH")) {
		if _, err := os.Stat(filepath.Join(dir, "sts")); err == nil {
			return "sts"
		}
	}
	return ""
}
