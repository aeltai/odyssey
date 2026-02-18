# Grafana Dashboard Patterns & Conversion Guide

Reference for anyone adding new dashboards to Odyssey or fine-tuning conversions.

## What Odyssey handles automatically

| Pattern | Example | What Odyssey does |
|---------|---------|-------------------|
| Built-in time variables | `[$__rate_interval]`, `[${__interval}]`, `[$__range]` | Replaced with your chosen interval (default `[5m]`) |
| Custom interval variables | `[$interval]`, `[$resolution]`, `[$__range_s]` | `[$var]` → `[interval]`; `$__range_s` → seconds (e.g. 300 for 5m) |
| Datasource filtering | Targets with `datasource: {type: "loki"}` etc. | Skipped (only Prometheus panels converted) |
| Variable label selectors | `instance="$host"`, `job=~"$job"` | Baked with Grafana default or `--variable` override; remaining unknowns removed |
| Regex variable selectors | `instance=~"^$Node$"`, `pod=~"$pod.*"` | Substituted if value known; otherwise removed |
| Chained variables | `instance=~"$host:$port"` | Removed if we don't have both values |
| Template variable parsing | `templating.list` in dashboard JSON | Defaults extracted from `current.value` / `current.text` |
| Uppercase PromQL functions | `SUM(...)`, `RATE(...)` | Lowercased |
| Metric namespace prefixes | `pg_up` → `postgresql_pg_up` | Auto-detected and rewritten |
| Counter `_total` stripping | `commands_total` → `commands` (by agent) | Matched via suffix fallback |
| Legacy `rows[]` layout | Old Grafana 3.x/4.x dashboards | Walked recursively |
| Modern `panels[]` layout | Grafana 7.x/8.x+ dashboards | Parsed normally |
| Nested row panels | `type: "row"` with child `panels[]` | Recursed into |
| `targets[].expr` | Standard Prometheus queries | Extracted |
| `targets[].query` | Alternative query field | Extracted as fallback |
| `options.queries[]` | Grafana 8+ panel options | Extracted |

## What needs user attention during conversion

### 1. Metric availability (most common issue)

**Problem**: The Grafana dashboard references metrics from an exporter that isn't deployed, or the STS agent isn't configured to scrape it.

**Symptoms**: `odyssey check` shows `[MISSING]` for most or all panels.

**Fix**:
- Deploy the required exporter (postgres_exporter, mysqld_exporter, nginx-exporter, etc.)
- Add `ad.stackstate.com` annotations to the pod for auto-discovery:
  ```yaml
  annotations:
    ad.stackstate.com/CONTAINER.check_names: '["openmetrics"]'
    ad.stackstate.com/CONTAINER.init_configs: '[{}]'
    ad.stackstate.com/CONTAINER.instances: '[{"prometheus_url":"http://%%host%%:PORT/metrics","namespace":"NAME","metrics":["*"]}]'
  ```
- Wait 2-3 minutes for metrics to flow, then re-run `odyssey check`

### 2. Metric prefix mismatch

**Problem**: The STS openmetrics agent prepends a namespace to all metrics (e.g. `postgresql_` prefix turns `pg_up` into `postgresql_pg_up`). Odyssey auto-detects the most common prefix, but some dashboards mix metrics from different exporters.

**Symptoms**: `odyssey check` shows some panels as AVAILABLE and some as MISSING, even though you know the exporter is running.

**Fix**:
- Run `odyssey check` first to see the auto-detected prefix
- If the prefix is wrong, override it: `--metric-prefix postgresql`
- If the dashboard mixes exporters (e.g. MySQL + node_exporter), the node_exporter metrics will show as missing — use `--include-missing` and accept those panels won't have data until node_exporter is configured

### 3. Counter metric `_total` suffix

**Problem**: The openmetrics agent strips `_total` from counter metrics. So `mysql_global_status_commands_total` in Grafana becomes `mysql_mysql_global_status_commands` in STS.

**Symptoms**: Panel shows `[MISSING]` for `*_total` metrics even though the exporter reports them.

**Fix**: Odyssey handles this automatically since v4abd4fb. If you're on an older version, update.

### 4. Custom `$interval` variable (not `$__interval`)

**Problem**: Many dashboards (especially MySQL, Redis) use a custom `$interval` variable instead of the built-in `$__interval`. This was inside range brackets like `[$interval]`.

**Symptoms**: Parse errors like `bad number or duration syntax: ""`.

**Fix**: Odyssey handles this automatically. All `[$variable]` patterns are replaced with your chosen interval (default 5m). Use `--interval 1m` (CLI) or the PromQL Interval dropdown (web UI) to change it.

### 5. Dashboard uses non-Prometheus datasource

**Problem**: Some panels query Loki, InfluxDB, Elasticsearch, or SQL instead of Prometheus. Odyssey only handles PromQL.

**Symptoms**: Panels with no `expr` field are silently skipped — you'll see fewer panels than expected.

**Fix**: This is expected. Odyssey only converts Prometheus/PromQL panels. Non-Prometheus panels need manual creation in STS.

### 6. Deprecated or removed metrics

**Problem**: Some Grafana dashboards reference metrics that no longer exist in newer exporter versions.

**Symptoms**: `[MISSING]` even though everything is deployed correctly.

**Known examples**:
| Metric | Status |
|--------|--------|
| `mysql_global_variables_innodb_additional_mem_pool_size` | Removed in MySQL 8.0 |
| `mysql_global_status_innodb_mem_dictionary` | Deprecated |
| `mysql_global_status_innodb_mem_adaptive_hash` | Deprecated |
| `node_cpu` (without `_seconds_total`) | Renamed in node_exporter v1.0 |

**Fix**: Use `--include-missing` to keep these panels (they'll show "No data"). Or find an updated Grafana dashboard revision.

### 7. Panel types don't affect queries (but affect display)

Odyssey converts all panels to `TimeSeriesChart` in STS. The original Grafana panel type (stat, gauge, bargauge, heatmap, table, etc.) is lost. This is a known limitation.

**Panel types found in popular dashboards**:

| Panel type | Used by | Notes |
|-----------|---------|-------|
| `graph` | Legacy (pre-8.x) | Most common in older dashboards |
| `timeseries` | Modern (8.x+) | Replacement for graph |
| `stat` | All versions | Single-value display |
| `gauge` | All versions | Gauge visualization |
| `bargauge` | 8.x+ | Bar gauge |
| `singlestat` | Legacy | Replaced by stat |
| `table` | All versions | Tabular data |
| `heatmap` | All versions | Heat map |

**Future improvement**: Map Grafana panel types to STS panel types (GaugeChart, etc.).

### 8. Units are not preserved

Grafana dashboards specify units (bytes, percent, ops/s, etc.) in `fieldConfig.defaults.unit`. Odyssey doesn't map these to STS panel units yet.

**Common units across popular dashboards**:
- `bytes`, `decbytes` — Memory, disk
- `percent`, `percentunit` — CPU, utilization
- `s`, `ms` — Latency
- `Bps`, `bps` — Network throughput
- `iops`, `ops` — Disk/request operations
- `short`, `none` — Generic numbers
- `hertz`, `celsius` — Hardware

**Future improvement**: Map `fieldConfig.defaults.unit` to STS `format.unit`.

### 9. Legend format templates

Grafana uses `legendFormat: "{{instance}}"` to label series. This is not carried into STS.

**Future improvement**: Map `legendFormat` to STS query aliases.

## Common dashboard structure patterns

### Pattern 1: Simple flat panels (most modern dashboards)

```json
{
  "panels": [
    {"type": "stat", "targets": [{"expr": "up"}]},
    {"type": "timeseries", "targets": [{"expr": "rate(x[5m])"}]}
  ]
}
```

**Odyssey**: Handled.

### Pattern 2: Row-based layout (legacy)

```json
{
  "rows": [
    {
      "title": "CPU",
      "panels": [
        {"type": "graph", "targets": [{"expr": "node_cpu"}]}
      ]
    }
  ]
}
```

**Odyssey**: Handled (walks `rows[]`).

### Pattern 3: Collapsed rows with nested panels

```json
{
  "panels": [
    {
      "type": "row",
      "title": "Memory",
      "collapsed": true,
      "panels": [
        {"type": "timeseries", "targets": [{"expr": "node_mem"}]}
      ]
    }
  ]
}
```

**Odyssey**: Handled (recurses into row panels).

### Pattern 4: Repeat panels

```json
{
  "type": "timeseries",
  "repeat": "namespace",
  "repeatDirection": "h",
  "maxPerRow": 4
}
```

**Odyssey**: The panel is converted once. The repeat-per-variable behavior is lost.

### Pattern 5: Mixed datasources

```json
{
  "datasource": {"uid": "-- Mixed --"},
  "targets": [
    {"datasource": {"type": "prometheus"}, "expr": "..."},
    {"datasource": {"type": "loki"}, "expr": "{job=\"app\"}"}
  ]
}
```

**Odyssey**: Only Prometheus `expr` targets are extracted. Loki/other targets are silently skipped.

## Checklist for adding a new dashboard

1. Download the JSON from Grafana.com: `curl -sL "https://grafana.com/api/dashboards/ID/revisions/LATEST/download" -o dashboard.json`
2. Run `odyssey check dashboard.json` — note the parse errors and missing metrics
3. If parse errors: check for new variable patterns not yet handled
4. Deploy the required exporter with proper annotations
5. Wait 2-3 minutes, re-run `odyssey check`
6. Run `odyssey convert -o dashboard.sts.yaml dashboard.json`
7. Review the YAML — check query expressions look valid
8. Apply: `sts dashboard apply --file dashboard.sts.yaml`
9. Check in STS UI — verify panels show data

## Variable patterns by popularity

Analyzed across 11 popular dashboards (1860, 315, 9628, 14031, 12708, 15314, 11835, 7589, 3070 + our test dashboards):

| Variable | Frequency | Used as |
|----------|-----------|---------|
| `$instance` | 9/11 | Label selector |
| `$job` | 6/11 | Label selector |
| `$__rate_interval` | 5/11 | Range duration |
| `$node` | 4/11 | Label selector |
| `$namespace` | 3/11 | Label selector |
| `$interval` (custom) | 3/11 | Range duration |
| `$host` | 2/11 | Label selector |
| `$release` | 2/11 | Label selector |
| `$__interval` | 2/11 | Range duration |
| `$datname` | 1/11 | Label selector |
| `$diskdevices` | 1/11 | Label selector |
| `$policy_name` | 1/11 | Label selector |
| `$topic` | 1/11 | Label selector |
| `$Node` | 1/11 | Label selector |
| `$__range` | rare | Range duration |
| `$resolution` | rare | Range duration |
