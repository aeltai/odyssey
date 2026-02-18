package sts

import "testing"

func TestMetricIndex_Has_ExactMatch(t *testing.T) {
	idx := &MetricIndex{
		Exact:  map[string]bool{"up": true, "node_cpu": true},
		Suffix: map[string]string{},
	}
	ok, actual := idx.Has("up")
	if !ok || actual != "up" {
		t.Errorf("Has(up) = %v, %q; want true, up", ok, actual)
	}
}

func TestMetricIndex_Has_SuffixMatch(t *testing.T) {
	idx := &MetricIndex{
		Exact:  map[string]bool{"postgresql_pg_up": true},
		Suffix: map[string]string{"pg_up": "postgresql_pg_up"},
	}
	ok, actual := idx.Has("pg_up")
	if !ok || actual != "postgresql_pg_up" {
		t.Errorf("Has(pg_up) = %v, %q; want true, postgresql_pg_up", ok, actual)
	}
}

func TestMetricIndex_Has_NoMatch(t *testing.T) {
	idx := &MetricIndex{
		Exact:  map[string]bool{"up": true},
		Suffix: map[string]string{},
	}
	ok, actual := idx.Has("nonexistent_metric")
	if ok {
		t.Errorf("Has(nonexistent) = true, %q; want false", actual)
	}
}

func TestMetricIndex_Has_ExactTakesPrecedence(t *testing.T) {
	idx := &MetricIndex{
		Exact:  map[string]bool{"pg_up": true, "postgresql_pg_up": true},
		Suffix: map[string]string{"pg_up": "postgresql_pg_up"},
	}
	ok, actual := idx.Has("pg_up")
	if !ok || actual != "pg_up" {
		t.Errorf("Has(pg_up) = %v, %q; want true, pg_up (exact preferred)", ok, actual)
	}
}

func TestLoadConfig_MissingBoth(t *testing.T) {
	t.Setenv("STS_URL", "")
	t.Setenv("STS_API_TOKEN", "")
	t.Setenv("HOME", t.TempDir())
	_, err := LoadConfig("", "")
	if err == nil {
		t.Error("expected error for missing URL+token, got nil")
	}
}

func TestLoadConfig_FromFlags(t *testing.T) {
	cfg, err := LoadConfig("https://sts.example.com", "tok123")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL != "https://sts.example.com" {
		t.Errorf("URL = %q", cfg.URL)
	}
	if cfg.Token != "tok123" {
		t.Errorf("Token = %q", cfg.Token)
	}
}

func TestLoadConfig_FromEnv(t *testing.T) {
	t.Setenv("STS_URL", "https://env.example.com")
	t.Setenv("STS_API_TOKEN", "envtok")
	cfg, err := LoadConfig("", "")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL != "https://env.example.com" {
		t.Errorf("URL = %q", cfg.URL)
	}
}

func TestLoadConfig_TrailingSlashTrimmed(t *testing.T) {
	cfg, err := LoadConfig("https://sts.example.com/", "tok")
	if err != nil {
		t.Fatal(err)
	}
	if cfg.URL != "https://sts.example.com" {
		t.Errorf("URL = %q, trailing slash not trimmed", cfg.URL)
	}
}
