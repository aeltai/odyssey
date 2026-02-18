package sts

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"os"
	"regexp"
	"strings"
)

// Config holds the SUSE Observability connection details.
type Config struct {
	URL   string
	Token string
}

// LoadConfig resolves STS URL and API token from explicit values, environment
// variables, or the sts CLI config file, in that priority order.
func LoadConfig(flagURL, flagToken string) (Config, error) {
	c := Config{URL: flagURL, Token: flagToken}
	if c.URL == "" {
		c.URL = os.Getenv("STS_URL")
	}
	if c.Token == "" {
		c.Token = os.Getenv("STS_API_TOKEN")
	}
	if c.URL == "" || c.Token == "" {
		fileURL, fileToken := readSTSConfig()
		if c.URL == "" {
			c.URL = fileURL
		}
		if c.Token == "" {
			c.Token = fileToken
		}
	}
	c.URL = strings.TrimRight(c.URL, "/")
	if c.URL == "" || c.Token == "" {
		return c, fmt.Errorf("STS URL and API token required (use flags, STS_URL/STS_API_TOKEN env, or sts context save)")
	}
	return c, nil
}

func readSTSConfig() (string, string) {
	home, _ := os.UserHomeDir()
	data, err := os.ReadFile(home + "/.config/stackstate-cli/config.yaml")
	if err != nil {
		return "", ""
	}
	content := string(data)
	urlRe := regexp.MustCompile(`url:\s*(\S+)`)
	tokenRe := regexp.MustCompile(`api-token:\s*(\S+)`)
	var url, token string
	if m := urlRe.FindStringSubmatch(content); len(m) > 1 {
		url = m[1]
	}
	if m := tokenRe.FindStringSubmatch(content); len(m) > 1 {
		token = m[1]
	}
	return url, token
}

// MetricIndex holds available metrics with both exact and suffix-based lookups.
// The openmetrics integration often prepends a namespace prefix (e.g. "postgresql_")
// to every metric, so pg_static becomes postgresql_pg_static. The suffix index
// lets us match the bare Grafana metric name against prefixed STS names.
type MetricIndex struct {
	Exact  map[string]bool
	Suffix map[string]string // bare name -> longest matching full name
}

// Has returns true if the metric exists exactly or via suffix match.
// On a match it also returns the actual STS metric name.
func (mi *MetricIndex) Has(name string) (bool, string) {
	if mi.Exact[name] {
		return true, name
	}
	if full, ok := mi.Suffix[name]; ok {
		return true, full
	}
	return false, ""
}

// FetchAvailableMetrics returns a MetricIndex from SUSE Observability's Prometheus API.
func FetchAvailableMetrics(cfg Config) (*MetricIndex, error) {
	endpoint := cfg.URL + "/prometheus/api/v1/label/__name__/values"
	req, err := http.NewRequest("GET", endpoint, nil)
	if err != nil {
		return nil, err
	}
	req.Header.Set("X-API-Token", cfg.Token)

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("fetch metrics from %s: %w", endpoint, err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != 200 {
		body, _ := io.ReadAll(resp.Body)
		return nil, fmt.Errorf("HTTP %d from %s: %s", resp.StatusCode, endpoint, string(body))
	}

	var result struct {
		Data []string `json:"data"`
	}
	if err := json.NewDecoder(resp.Body).Decode(&result); err != nil {
		return nil, fmt.Errorf("decode metrics response: %w", err)
	}

	idx := &MetricIndex{
		Exact:  make(map[string]bool, len(result.Data)),
		Suffix: make(map[string]string, len(result.Data)),
	}
	for _, n := range result.Data {
		idx.Exact[n] = true
		for i := 0; i < len(n); i++ {
			if n[i] == '_' && i+1 < len(n) {
				suffix := n[i+1:]
				if existing, ok := idx.Suffix[suffix]; !ok || len(n) > len(existing) {
					idx.Suffix[suffix] = n
				}
			}
		}
	}
	return idx, nil
}
