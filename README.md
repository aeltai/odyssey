# Odyssey

Convert Grafana dashboards to [SUSE Observability](https://www.suse.com/products/observability/) (StackState) dashboard YAML — with an interactive wizard or non-interactive CLI.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/License-Apache_2.0-blue" />
</p>

## Features

- **Web UI** — run `odyssey server` for a browser-based dashboard conversion experience
- **Interactive wizard** — run `odyssey` with no arguments for a guided terminal experience
- **Non-interactive CLI** — `odyssey convert` and `odyssey check` for scripting and CI
- Handles all common Grafana JSON layouts: `panels[]`, `rows[]`, nested panels, `targets[].expr`, and `options.queries[]`
- Sanitises Grafana-specific PromQL: template variables, time-range variables, uppercase functions
- **Configurable interval** — replace `$__rate_interval`, `$__interval`, `$__range_s` with your chosen duration (default 5m; use `--interval` or web UI)
- **Variable bake-in** — Grafana template variables (e.g. `$namespace`, `$job`) are parsed from `templating.list` and their default values are baked into queries. Override via `--variable namespace=prod` (CLI) or the web UI
- Auto-detects metric namespace prefixes (e.g. `pg_up` → `postgresql_pg_up`) and rewrites queries
- Merges multiple Grafana JSON files into a single STS dashboard
- `check` mode prints a per-panel metric-availability report

## Install

```bash
go install github.com/aeltai/odyssey@latest
```

Or build from source:

```bash
git clone https://github.com/aeltai/odyssey.git
cd odyssey
make build    # produces ./odyssey
```

## Quick start

### Web UI

```bash
odyssey server
# → http://localhost:3000
```

Upload Grafana JSON files, configure your STS connection, preview panel analysis, and download the converted YAML — all from your browser.

### Interactive mode (terminal)

```bash
odyssey
```

The wizard will guide you through:
1. Selecting a Grafana dashboard JSON file
2. Connecting to your SUSE Observability instance
3. Reviewing metric availability
4. Configuring the output dashboard
5. Generating YAML and optionally applying it

### Non-interactive mode

```bash
# Check which panels would have data
odyssey check postgres-dashboard.json

# Convert a dashboard
odyssey convert -o postgres.sts.yaml postgres-dashboard.json

# Include all panels even if metrics are missing
odyssey convert --include-missing -o full.sts.yaml dashboard.json

# Bake namespace and job variables into queries
odyssey convert -v namespace=prod -v job=api-server -o filtered.sts.yaml dashboard.json

# Merge multiple dashboards
odyssey convert --name "All Services" -o merged.sts.yaml nginx.json mysql.json

# Apply the result
sts dashboard apply --file postgres.sts.yaml
```

## Commands

### `odyssey` (no subcommand)

Launches the interactive wizard.

### `odyssey convert`

```
odyssey convert [flags] <dashboard.json> [more.json ...]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-o`, `--output` | `<input>.sts.yaml` | Output YAML file path |
| `--name` | from Grafana title | Dashboard name in STS |
| `--interval` | `5m` | PromQL interval for rate/irate (e.g. 1m, 5m, 15m, 1h) |
| `-v`, `--variable` | *(none)* | Bake variable value into queries (repeatable, e.g. `-v namespace=prod`) |
| `--include-missing` | `false` | Include panels with missing metrics |
| `--rewrite-metrics` | `true` | Rewrite queries with detected prefix |
| `--metric-prefix` | auto-detected | Override the namespace prefix |
| `--id` | `0` (create new) | Existing STS dashboard ID for updates |
| `--sts-url` | | SUSE Observability base URL |
| `--sts-token` | | SUSE Observability API token |

### `odyssey check`

```
odyssey check [flags] <dashboard.json> [more.json ...]
```

| Flag | Default | Description |
|------|---------|-------------|
| `--sts-url` | | SUSE Observability base URL |
| `--sts-token` | | SUSE Observability API token |
| `--interval` | `5m` | PromQL interval for rate/irate (e.g. 1m, 5m, 15m, 1h) |
| `-v`, `--variable` | *(none)* | Bake variable value into queries (repeatable, e.g. `-v namespace=prod`) |

### `odyssey server`

```
odyssey server [flags]
```

| Flag | Default | Description |
|------|---------|-------------|
| `-p`, `--port` | `3000` | Port to listen on |

Starts the web UI. The frontend is embedded in the binary — no Node.js required at runtime.

### `odyssey version`

Print the version and exit.

## Configuration

The STS connection is resolved in priority order:

| Source | URL | Token |
|--------|-----|-------|
| CLI flags | `--sts-url` | `--sts-token` |
| Environment | `STS_URL` | `STS_API_TOKEN` |
| sts CLI config | `~/.config/stackstate-cli/config.yaml` | (same) |

## How it works

```
┌─────────────────┐    ┌──────────────┐    ┌────────────────┐
│  Grafana JSON(s) │───>│ Parse panels │───>│ Sanitise PromQL│
└─────────────────┘    └──────────────┘    └───────┬────────┘
                                                   │
                       ┌──────────────┐    ┌───────▼────────┐
                       │ STS Prom API │───>│ Match metrics  │
                       └──────────────┘    └───────┬────────┘
                                                   │
                       ┌──────────────┐    ┌───────▼────────┐
                       │  YAML output │<───│ Build dashboard│
                       └──────────────┘    └────────────────┘
```

1. **Parse** — Walk all Grafana panels (including rows, nested panels) and extract every PromQL expression
2. **Sanitise** — Bake Grafana template-variable values into label selectors (from `templating.list` defaults or `--variable` overrides), remove remaining variable references, replace `$__rate_interval` / `$__interval` / `$__range_s` with your chosen interval (default 5m), lowercase function names
3. **Extract** — Use the Prometheus PromQL parser to identify all metric names
4. **Fetch** — Call `GET {url}/prometheus/api/v1/label/__name__/values` to get available STS metrics
5. **Match** — Check each panel's metrics with prefix-aware suffix matching
6. **Rewrite** — Update metric names in queries to use the detected agent prefix
7. **Build** — Generate STS dashboard YAML (`dashboard.spec.layouts` + `dashboard.spec.panels`)

## PromQL sanitisation rules

| Grafana pattern | Replacement |
|----------------|-------------|
| `$__rate_interval`, `$__interval`, `$__range` | Your chosen interval (default `5m`) |
| `$__range_s` | Seconds derived from interval (e.g. 300 for 5m) |
| `label="$variable"` (with override) | `label="value"` — baked from Grafana default or `--variable` |
| `label="$variable"` (no override) | *(removed)* |
| `label=~"$variable"` (with override) | `label=~"value"` |
| `label=~"$variable"` (no override) | *(removed)* |
| `SUM(...)`, `AVG(...)` | `sum(...)`, `avg(...)` |

Use `--interval 1m` (CLI) or the interval selector (web UI) to change the default. Use `-v namespace=prod` to bake variable values into queries.

## Verified dashboards

Dashboards in [`examples/`](examples/) have been deployed to a live SUSE Observability instance and verified end-to-end.

| Dashboard | Grafana ID | Panels | Verified | Notes |
|-----------|-----------|--------|----------|-------|
| PostgreSQL | [9628](https://grafana.com/grafana/dashboards/9628) | 40 (34 live) | Yes | Prefix `postgresql` auto-rewritten |
| NGINX Ingress | [12708](https://grafana.com/grafana/dashboards/12708) | 8 (8 live) | Yes | 100% coverage |
| MySQL Overview | [14031](https://grafana.com/grafana/dashboards/14031) | 25 (21 live) | Yes | 4 use deprecated MySQL 8.0 metrics |
| Kubewarden | [15760](https://grafana.com/grafana/dashboards/15760) | 24 | Yes | Requires Kubewarden metrics |

### Integration-tested dashboards (47)

The full test suite validates parsing, sanitisation, and YAML generation against the **top 47 Grafana dashboards** (1,551 panels total). Run with `make test-integration`.

## Development

```bash
make test              # run all tests with race detector
make test-integration  # download 47 dashboards and run integration tests
make lint              # go vet
make build             # build binary
make install           # go install
make clean             # remove binary
```

### Project structure

```
odyssey/
├── main.go                        # Entry point
├── cmd/
│   ├── root.go                    # Cobra root command
│   ├── server.go                  # Web UI server (embedded Vue)
│   ├── convert.go                 # Non-interactive convert
│   ├── check.go                   # Non-interactive check
│   ├── interactive.go             # Interactive wizard (huh)
│   ├── exec.go                    # Command execution helper
│   └── dist/                      # Built frontend (auto-generated)
├── web/                           # Vue 3 + Vite + Tailwind frontend
│   ├── src/
│   │   ├── App.vue                # Main app with wizard stepper
│   │   └── components/
│   │       ├── StepUpload.vue     # Drag & drop file upload
│   │       ├── StepConfig.vue     # STS connection + options
│   │       ├── StepResults.vue    # Panel analysis & metrics
│   │       └── StepOutput.vue     # YAML preview & download
│   ├── vite.config.js
│   └── package.json
├── internal/
│   ├── api/
│   │   └── handler.go             # HTTP API handlers
│   ├── engine/
│   │   └── engine.go              # Shared conversion logic
│   ├── integration_test.go        # 47-dashboard integration tests
│   ├── grafana/
│   │   ├── parse.go               # Grafana JSON parser
│   │   └── parse_test.go
│   ├── promql/
│   │   ├── metrics.go             # PromQL metric extraction
│   │   └── metrics_test.go
│   └── sts/
│       ├── client.go              # STS API client + MetricIndex
│       ├── client_test.go
│       ├── dashboard.go           # STS YAML builder
│       ├── dashboard_test.go
│       ├── sanitize.go            # PromQL sanitisation
│       └── sanitize_test.go
├── examples/
│   ├── grafana-json/              # Source Grafana dashboards
│   └── verified/                  # Converted STS YAML (ready to apply)
├── testdata/dashboards/           # 47 dashboards (downloaded via script)
├── scripts/download-testdata.sh
├── Makefile
├── go.mod / go.sum
└── .gitignore
```

## License

Apache License 2.0
