# Verified Dashboard Examples

This directory contains Grafana dashboards that have been **fully verified**
through the odyssey migration pipeline and deployed to a live
SUSE Observability instance.

## Verified Dashboards

| Dashboard | Grafana ID | Panels | Verified | Notes |
|-----------|-----------|--------|----------|-------|
| **PostgreSQL** | [9628](https://grafana.com/grafana/dashboards/9628) | 40 (34 with live data) | Yes | Metric prefix `postgresql` auto-detected and rewritten |
| **NGINX Ingress** | [12708](https://grafana.com/grafana/dashboards/12708) | 8 (8 with live data) | Yes | 100% panel coverage |
| **MySQL Overview** | [14031](https://grafana.com/grafana/dashboards/14031) | 25 (21 with live data) | Yes | 4 panels use deprecated MySQL 8.0 metrics |
| **Kubewarden** | [15760](https://grafana.com/grafana/dashboards/15760) | 24 (converted) | Yes | Requires Kubewarden controller with metrics enabled |

## Directory Structure

```
examples/
  grafana-json/          # Original Grafana dashboard JSON files
    postgresql.json
    nginx-ingress.json
    mysql-overview.json
    kubewarden.json
  verified/              # Converted STS YAML (ready to apply)
    postgresql.sts.yaml
    nginx-ingress.sts.yaml
    mysql-overview.sts.yaml
    kubewarden.sts.yaml
```

## How to Use

Apply any verified dashboard directly:

```bash
sts dashboard apply --file examples/verified/postgresql.sts.yaml
```

Or re-run the conversion yourself to check against your own metrics:

```bash
odyssey convert examples/grafana-json/postgresql.json \
  --sts-url https://your-instance.sslip.io \
  --sts-token YOUR_TOKEN \
  --metric-prefix postgresql \
  --rewrite-metrics
```

## Verification Environment

These dashboards were tested on:

- **Platform**: RKE2 cluster on AWS (3 nodes)
- **SUSE Observability**: v2.x with StackState agent (auto-discovery via pod annotations)
- **Exporters**: postgres_exporter, nginx-exporter (prometheus-community), mysqld-exporter v0.15.1
- **Odyssey version**: v0.1.0+

## Why Some Panels Show "Missing"

Panels marked as missing during `odyssey check` fall into these categories:

1. **Metric not scraped** — the exporter is not deployed or annotations are missing
2. **Deprecated metrics** — MySQL 8.0 removed some `mysql_global_status_*` metrics
3. **Agent prefix mismatch** — STS agent adds a prefix (e.g., `postgresql_`) that needs `--rewrite-metrics`
4. **Counter suffix stripping** — OpenMetrics agent strips `_total` from counters; odyssey handles this automatically
