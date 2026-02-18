package api

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/aeltai/odyssey/internal/engine"
	"github.com/aeltai/odyssey/internal/grafana"
	"github.com/aeltai/odyssey/internal/sts"
	"gopkg.in/yaml.v3"
)

type ParseRequest struct {
	Filename string          `json:"filename"`
	Content  json.RawMessage `json:"content"`
}

type ParseResponse struct {
	Title  string        `json:"title"`
	Panels []PanelResult `json:"panels"`
}

type PanelResult struct {
	Title     string   `json:"title"`
	Expr      string   `json:"expr"`
	Original  string   `json:"original,omitempty"`
	Sanitized string   `json:"sanitized"`
	Metrics   []string `json:"metrics"`
	HasData   bool     `json:"hasData"`
	Warning   string   `json:"warning,omitempty"`
}

type CheckRequest struct {
	Dashboards []ParseRequest `json:"dashboards"`
	STSURL     string         `json:"stsUrl"`
	STSToken   string         `json:"stsToken"`
}

type CheckResponse struct {
	Panels         []PanelResult `json:"panels"`
	TotalMetrics   int           `json:"totalMetrics"`
	Matched        int           `json:"matched"`
	Missing        int           `json:"missing"`
	DetectedPrefix string        `json:"detectedPrefix"`
}

type ConvertRequest struct {
	Dashboards     []ParseRequest `json:"dashboards"`
	STSURL         string         `json:"stsUrl"`
	STSToken       string         `json:"stsToken"`
	Name           string         `json:"name"`
	MetricPrefix   string         `json:"metricPrefix"`
	RewriteMetrics bool           `json:"rewriteMetrics"`
	IncludeMissing bool           `json:"includeMissing"`
}

type ConvertResponse struct {
	YAML           string `json:"yaml"`
	PanelCount     int    `json:"panelCount"`
	Matched        int    `json:"matched"`
	Missing        int    `json:"missing"`
	DetectedPrefix string `json:"detectedPrefix"`
}

type ApplyRequest struct {
	YAML     string `json:"yaml"`
	STSURL   string `json:"stsUrl"`
	STSToken string `json:"stsToken"`
}

type ApplyResponse struct {
	Success     bool   `json:"success"`
	DashboardID int64  `json:"dashboardId,omitempty"`
	Name        string `json:"name,omitempty"`
	Message     string `json:"message"`
}

type GrafanaConnectRequest struct {
	URL   string `json:"url"`
	Token string `json:"token"`
}

type GrafanaDashboardSummary struct {
	UID   string `json:"uid"`
	Title string `json:"title"`
	URI   string `json:"uri,omitempty"`
	Type  string `json:"type,omitempty"`
	Tags  []string `json:"tags,omitempty"`
}

type GrafanaDashboardFetchRequest struct {
	URL   string `json:"url"`
	Token string `json:"token"`
	UID   string `json:"uid"`
}

func NewRouter() http.Handler {
	mux := http.NewServeMux()
	mux.HandleFunc("POST /api/parse", handleParse)
	mux.HandleFunc("POST /api/check", handleCheck)
	mux.HandleFunc("POST /api/convert", handleConvert)
	mux.HandleFunc("POST /api/apply", handleApply)
	mux.HandleFunc("POST /api/grafana/dashboards", handleGrafanaDashboards)
	mux.HandleFunc("POST /api/grafana/dashboard", handleGrafanaDashboard)
	mux.HandleFunc("GET /api/health", handleHealth)
	return corsMiddleware(mux)
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func handleHealth(w http.ResponseWriter, r *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func handleParse(w http.ResponseWriter, r *http.Request) {
	var req ParseRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	title, panels, err := parseFromJSON(req.Content)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var buf bytes.Buffer
	enriched := engine.SanitiseAndExtract(panels, &buf)

	results := make([]PanelResult, 0, len(enriched))
	for _, ep := range enriched {
		pr := PanelResult{
			Title:     ep.Panel.Title,
			Expr:      ep.Panel.Expr,
			Original:  ep.Panel.Expr,
			Sanitized: ep.Sanitized,
			Metrics:   ep.MetricNames,
		}
		if ep.ParseError {
			pr.Warning = "PromQL parse error"
		}
		if pr.Metrics == nil {
			pr.Metrics = []string{}
		}
		results = append(results, pr)
	}

	writeJSON(w, http.StatusOK, ParseResponse{Title: title, Panels: results})
}

func handleCheck(w http.ResponseWriter, r *http.Request) {
	var req CheckRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	panels, _, err := parseDashboards(req.Dashboards)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var buf bytes.Buffer
	enriched := engine.SanitiseAndExtract(panels, &buf)

	stsCfg, err := sts.LoadConfig(req.STSURL, req.STSToken)
	if err != nil {
		writeErr(w, http.StatusBadRequest, "STS connection: "+err.Error())
		return
	}

	metricIdx, err := sts.FetchAvailableMetrics(stsCfg)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "fetch metrics: "+err.Error())
		return
	}

	matchResults, detectedPrefix := engine.MatchPanels(enriched, metricIdx)
	matched, missing := engine.CountResults(matchResults)

	results := make([]PanelResult, 0, len(matchResults))
	for i, mr := range matchResults {
		pr := PanelResult{
			Title:     mr.Title,
			Expr:      panels[i].Expr,
			Original:  panels[i].Expr,
			Sanitized: mr.Expr,
			Metrics:   mr.Metrics,
			HasData:   mr.HasData,
		}
		if mr.ParseError {
			pr.Warning = "PromQL parse error"
		}
		if pr.Metrics == nil {
			pr.Metrics = []string{}
		}
		results = append(results, pr)
	}

	writeJSON(w, http.StatusOK, CheckResponse{
		Panels:         results,
		TotalMetrics:   len(metricIdx.Exact),
		Matched:        matched,
		Missing:        missing,
		DetectedPrefix: detectedPrefix,
	})
}

func handleConvert(w http.ResponseWriter, r *http.Request) {
	var req ConvertRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}

	panels, title, err := parseDashboards(req.Dashboards)
	if err != nil {
		writeErr(w, http.StatusUnprocessableEntity, err.Error())
		return
	}

	var buf bytes.Buffer
	enriched := engine.SanitiseAndExtract(panels, &buf)

	var matchResults []engine.MatchResult
	var detectedPrefix string
	var metricIdx *sts.MetricIndex

	if req.STSURL != "" && req.STSToken != "" {
		stsCfg, err := sts.LoadConfig(req.STSURL, req.STSToken)
		if err != nil {
			writeErr(w, http.StatusBadRequest, "STS connection: "+err.Error())
			return
		}
		metricIdx, err = sts.FetchAvailableMetrics(stsCfg)
		if err != nil {
			writeErr(w, http.StatusBadGateway, "fetch metrics: "+err.Error())
			return
		}
		matchResults, detectedPrefix = engine.MatchPanels(enriched, metricIdx)
	}

	var panelInputs []sts.PanelInput
	matched, missing := 0, 0

	if matchResults != nil {
		matched, missing = engine.CountResults(matchResults)
		panelInputs = engine.BuildPanelInputs(matchResults, req.IncludeMissing)
	} else {
		for _, ep := range enriched {
			panelInputs = append(panelInputs, sts.PanelInput{
				Title: ep.Panel.Title,
				Expr:  ep.Sanitized,
			})
		}
		matched = len(panelInputs)
	}

	prefix := req.MetricPrefix
	if prefix == "" {
		prefix = detectedPrefix
	}
	if req.RewriteMetrics && prefix != "" && metricIdx != nil {
		for i, pi := range panelInputs {
			panelInputs[i].Expr = sts.RewriteMetricPrefix(pi.Expr, prefix, metricIdx)
		}
	}

	name := req.Name
	if name == "" {
		name = title
	}
	if name == "" {
		name = "Grafana migrated"
	}

	dash := sts.BuildDashboard(name, "publicDashboard", 0, panelInputs)

	var yamlBuf bytes.Buffer
	enc := yaml.NewEncoder(&yamlBuf)
	enc.SetIndent(2)
	if err := enc.Encode(dash); err != nil {
		writeErr(w, http.StatusInternalServerError, "generate YAML: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, ConvertResponse{
		YAML:           yamlBuf.String(),
		PanelCount:     len(panelInputs),
		Matched:        matched,
		Missing:        missing,
		DetectedPrefix: detectedPrefix,
	})
}

// handleApply sends the generated dashboard YAML to the STS Dashboards API.
func handleApply(w http.ResponseWriter, r *http.Request) {
	var req ApplyRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.STSURL == "" || req.STSToken == "" {
		writeErr(w, http.StatusBadRequest, "STS URL and API token are required to apply")
		return
	}
	if req.YAML == "" {
		writeErr(w, http.StatusBadRequest, "no YAML to apply")
		return
	}

	var dashboardData map[string]interface{}
	if err := yaml.Unmarshal([]byte(req.YAML), &dashboardData); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid YAML: "+err.Error())
		return
	}

	jsonBody, err := json.Marshal(dashboardData)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "json marshal: "+err.Error())
		return
	}

	apiURL := strings.TrimRight(req.STSURL, "/") + "/api/dashboards"

	httpReq, err := http.NewRequest("POST", apiURL, bytes.NewReader(jsonBody))
	if err != nil {
		writeErr(w, http.StatusInternalServerError, "create request: "+err.Error())
		return
	}
	httpReq.Header.Set("Content-Type", "application/json")
	httpReq.Header.Set("X-API-Token", req.STSToken)

	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "STS API call failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	body, _ := io.ReadAll(resp.Body)

	if resp.StatusCode >= 400 {
		msg := string(body)
		if len(msg) > 500 {
			msg = msg[:500]
		}
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("STS returned %d: %s", resp.StatusCode, msg))
		return
	}

	var result map[string]interface{}
	json.Unmarshal(body, &result)

	applyResp := ApplyResponse{
		Success: true,
		Message: "Dashboard applied successfully",
	}
	if id, ok := result["id"].(float64); ok {
		applyResp.DashboardID = int64(id)
	}
	if name, ok := result["name"].(string); ok {
		applyResp.Name = name
	}

	writeJSON(w, http.StatusOK, applyResp)
}

// handleGrafanaDashboards lists dashboards from a Grafana instance.
func handleGrafanaDashboards(w http.ResponseWriter, r *http.Request) {
	var req GrafanaConnectRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.URL == "" {
		writeErr(w, http.StatusBadRequest, "Grafana URL is required")
		return
	}

	apiURL := strings.TrimRight(req.URL, "/") + "/api/search?type=dash-db&limit=200"

	httpReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpReq.Header.Set("Accept", "application/json")
	if req.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "Grafana connection failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		msg := string(body)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("Grafana returned %d: %s", resp.StatusCode, msg))
		return
	}

	var rawDashboards []struct {
		UID   string   `json:"uid"`
		Title string   `json:"title"`
		URI   string   `json:"uri"`
		Type  string   `json:"type"`
		Tags  []string `json:"tags"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&rawDashboards); err != nil {
		writeErr(w, http.StatusBadGateway, "failed to parse Grafana response: "+err.Error())
		return
	}

	dashboards := make([]GrafanaDashboardSummary, 0, len(rawDashboards))
	for _, d := range rawDashboards {
		dashboards = append(dashboards, GrafanaDashboardSummary{
			UID:   d.UID,
			Title: d.Title,
			URI:   d.URI,
			Type:  d.Type,
			Tags:  d.Tags,
		})
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"dashboards": dashboards,
		"total":      len(dashboards),
	})
}

// handleGrafanaDashboard fetches a single dashboard JSON from Grafana by UID.
func handleGrafanaDashboard(w http.ResponseWriter, r *http.Request) {
	var req GrafanaDashboardFetchRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		writeErr(w, http.StatusBadRequest, "invalid request: "+err.Error())
		return
	}
	if req.URL == "" || req.UID == "" {
		writeErr(w, http.StatusBadRequest, "Grafana URL and dashboard UID are required")
		return
	}

	apiURL := strings.TrimRight(req.URL, "/") + "/api/dashboards/uid/" + req.UID

	httpReq, err := http.NewRequest("GET", apiURL, nil)
	if err != nil {
		writeErr(w, http.StatusInternalServerError, err.Error())
		return
	}
	httpReq.Header.Set("Accept", "application/json")
	if req.Token != "" {
		httpReq.Header.Set("Authorization", "Bearer "+req.Token)
	}

	client := &http.Client{Timeout: 15 * time.Second}
	resp, err := client.Do(httpReq)
	if err != nil {
		writeErr(w, http.StatusBadGateway, "Grafana connection failed: "+err.Error())
		return
	}
	defer resp.Body.Close()

	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		msg := string(body)
		if len(msg) > 300 {
			msg = msg[:300]
		}
		writeErr(w, http.StatusBadGateway, fmt.Sprintf("Grafana returned %d: %s", resp.StatusCode, msg))
		return
	}

	var wrapper struct {
		Dashboard json.RawMessage `json:"dashboard"`
		Meta      struct {
			Slug string `json:"slug"`
		} `json:"meta"`
	}
	body, _ := io.ReadAll(resp.Body)
	if err := json.Unmarshal(body, &wrapper); err != nil {
		writeErr(w, http.StatusBadGateway, "failed to parse Grafana dashboard: "+err.Error())
		return
	}

	writeJSON(w, http.StatusOK, map[string]interface{}{
		"dashboard": json.RawMessage(wrapper.Dashboard),
		"slug":      wrapper.Meta.Slug,
	})
}

func parseDashboards(dbs []ParseRequest) ([]grafana.Panel, string, error) {
	var all []grafana.Panel
	var title string
	for _, db := range dbs {
		t, panels, err := parseFromJSON(db.Content)
		if err != nil {
			return nil, "", err
		}
		if title == "" && t != "" {
			title = t
		}
		all = append(all, panels...)
	}
	return all, title, nil
}

func parseFromJSON(raw json.RawMessage) (string, []grafana.Panel, error) {
	return grafana.ParseBytes(raw)
}

func writeJSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	json.NewEncoder(w).Encode(v)
}

func writeErr(w http.ResponseWriter, code int, msg string) {
	writeJSON(w, code, map[string]string{"error": msg})
}

// ServeEmbedded serves the embedded frontend files, falling back to index.html for SPA routing.
func ServeEmbedded(fs http.FileSystem) http.Handler {
	fileServer := http.FileServer(fs)
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		if strings.HasPrefix(path, "/api/") {
			http.NotFound(w, r)
			return
		}
		f, err := fs.Open(path)
		if err != nil {
			r.URL.Path = "/"
		} else {
			f.Close()
		}
		fileServer.ServeHTTP(w, r)
	})
}

// Drain reads and discards the request body.
func Drain(r *http.Request) {
	io.Copy(io.Discard, r.Body)
	r.Body.Close()
}
