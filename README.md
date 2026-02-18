# Odyssey

Convert Grafana dashboards to [SUSE Observability](https://www.suse.com/products/observability/) (StackState) dashboard YAML — with an interactive wizard or non-interactive CLI.

<p align="center">
  <img src="https://img.shields.io/badge/Go-1.22+-00ADD8?logo=go&logoColor=white" />
  <img src="https://img.shields.io/badge/License-Apache_2.0-blue" />
</p>

## Features

- **Interactive wizard** — run `odyssey` with no arguments for a guided experience
- **Non-interactive CLI** — `odyssey convert` and `odyssey check` for scripting and CI
- Handles all common Grafana JSON layouts: `panels[]`, `rows[]`, nested panels, `targets[].expr`, and `options.queries[]`
- Sanitises Grafana-specific PromQL: template variables, time-range variables, uppercase functions
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

### Interactive mode

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
2. **Sanitise** — Remove Grafana template-variable label selectors, replace `$__rate_interval` / `$__interval` with `5m`, lowercase function names
3. **Extract** — Use the Prometheus PromQL parser to identify all metric names
4. **Fetch** — Call `GET {url}/prometheus/api/v1/label/__name__/values` to get available STS metrics
5. **Match** — Check each panel's metrics with prefix-aware suffix matching
6. **Rewrite** — Update metric names in queries to use the detected agent prefix
7. **Build** — Generate STS dashboard YAML (`dashboard.spec.layouts` + `dashboard.spec.panels`)

## PromQL sanitisation rules

| Grafana pattern | Replacement |
|----------------|-------------|
| `$__rate_interval` | `5m` |
| `$__interval` | `5m` |
| `label="$variable"` | *(removed)* |
| `label=~"$variable"` | *(removed)* |
| `SUM(...)`, `AVG(...)` | `sum(...)`, `avg(...)` |

## Tested dashboards

| Dashboard | Grafana ID | Panels |
|-----------|-----------|--------|
| PostgreSQL | 9628 | 40 |
| NGINX | 12708 | 8 |
| MySQL Overview | 14031 | 25 |
| Kubewarden | 15314 | 24 |
| Kubernetes Cluster | 315 | 33 |

## Development

```bash
make test       # run tests with race detector
make lint       # go vet
make build      # build binary
make install    # go install
make clean      # remove binary
```

### Project structure

```
odyssey/
├── main.go                       # Entry point
├── cmd/
│   ├── root.go                   # Cobra root command
│   ├── convert.go                # Non-interactive convert
│   ├── check.go                  # Non-interactive check
│   ├── interactive.go            # Interactive wizard (huh)
│   └── exec.go                   # Command execution helper
├── internal/
│   ├── engine/
│   │   └── engine.go             # Shared conversion logic
│   ├── grafana/
│   │   ├── parse.go              # Grafana JSON parser
│   │   └── parse_test.go
│   ├── promql/
│   │   ├── metrics.go            # PromQL metric extraction
│   │   └── metrics_test.go
│   └── sts/
│       ├── client.go             # STS API client + MetricIndex
│       ├── client_test.go
│       ├── dashboard.go          # STS YAML builder
│       ├── dashboard_test.go
│       ├── sanitize.go           # PromQL sanitisation
│       └── sanitize_test.go
├── Makefile
├── go.mod / go.sum
└── .gitignore
```

## License

Apache License 2.0
