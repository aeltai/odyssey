# Parser Improvement Guide

Recommendations for making Odyssey's Grafana parsing smarter, covering edge cases, and aligning with standards.

## References & Standards

| Resource | URL | Use |
|----------|-----|-----|
| **Grafana JSON Model** | [grafana.com/docs/.../view-dashboard-json-model](https://grafana.com/docs/grafana/latest/dashboards/build-dashboards/view-dashboard-json-model/) | Canonical structure |
| **Prometheus Template Variables** | [grafana.com/docs/.../prometheus/template-variables](https://grafana.com/docs/grafana/latest/datasources/prometheus/template-variables/) | All variable syntax |
| **Grafana Schema v2** | [grafana.com/docs/.../schema-v2](https://grafana.com/docs/grafana/latest/as-code/observability-as-code/schema-v2/) | New layout (elements, layout) |
| **Dashboard Spec (7.0)** | [github.com/grafana/dashboard-spec](https://github.com/grafana/dashboard-spec) | Legacy schema (deprecated but useful) |
| **Prometheus rate() best practices** | [Grafana 7.2 blog: $__rate_interval](https://grafana.com/blog/new-in-grafana-7.2-rate-interval-for-prometheus-rate-queries-that-just-work/) | Why 5m fallback is sane |

---

## 1. Datasource Filtering

**Problem**: Panels can query Loki, InfluxDB, Elasticsearch, MySQL, etc. We only want PromQL.

**Current**: We extract any `targets[].expr` or `query` — non-Prometheus panels often have empty/irrelevant content.

**Improvement**:
- Check `target.datasource` — can be `null` (default), `"Prometheus"`, or `{"type":"prometheus","uid":"xyz"}`
- Skip targets where `datasource` explicitly points to non-Prometheus
- Log skipped panels for transparency

```go
// In rawTarget, add:
Datasource *json.RawMessage `json:"datasource"`

// When extracting: only add expr if datasource is Prometheus or nil
```

---

## 2. Additional Grafana Variables

**Documented variables we don't yet handle explicitly** (from [Prometheus template variables](https://grafana.com/docs/grafana/latest/datasources/prometheus/template-variables/)):

| Variable | Meaning | Replacement |
|----------|---------|--------------|
| `$__range` | Full dashboard time range (e.g. 6h) | User-chosen interval (default 5m) |
| `$__range_s` | Range in seconds | Seconds derived from interval (e.g. 300 for 5m) |
| `$__from` / `$__to` | Epoch milliseconds | Remove or replace — rarely in PromQL |
| `$resolution` | Step / resolution | `5m` |
| `$__rate_interval` | ✅ Already handled | `5m` |
| `$__interval` | ✅ Already handled | `5m` |

**Implemented** (with user-configurable interval via `--interval` / web UI):
```go
// $__range and $__range_s — replaced with user interval (default 5m)
// $__range_s uses time.ParseDuration to derive seconds from interval
```

---

## 3. Query Location Edge Cases

**Current extraction** checks:
- `targets[].expr`
- `targets[].query` (fallback)
- `options.queries[]` (Grafana 8+)

**Potential additions**:
- **Grafana 9+ / Schema v2**: `elements[].data.queries[]` or layout-based structure
- **Mixed datasource panels**: `targets` may have mixed types; filter by datasource
- **Transformations**: `transformations` array — not PromQL, ignore
- **Expression queries**: Some panels use `expressions` for math; might contain PromQL refs (advanced)

---

## 4. Panel Structure Variants

| Layout | Structure | Current support |
|--------|-----------|-----------------|
| Legacy `rows[]` | Top-level rows array | ✅ Yes |
| `panels[]` flat | Panels at top level | ✅ Yes |
| `panels[]` nested | Row panels with `panels` children | ✅ Yes |
| **Tabs** | `panels[].panels[]` with type `"tab"` or similar | ⚠️ Partial (we walk recursively) |
| **Schema v2** | `elements`, `layout` with different structure | ❌ No |
| **Collapsed rows** | `panels` might be `null` when row collapsed in editor | ⚠️ Could skip — export usually expands |

**Recommendation**: Add defensive checks for `panels == nil` and `len(panels) == 0` in row handling. Schema v2 can be a future phase.

---

## 5. Variable Label Selector Edge Cases

**Current** `varLabelFilter` removes:
- `job=~"$job"`, `namespace="$namespace"`, `instance="$node"`

**Possible gaps**:
- **Multi-value expansion**: `$node` with "All" might become `(a|b|c)` — our filter may leave `(a|b|c)` in place; Prometheus accepts it if the var was substituted at export, but raw `$var` we remove
- **Nested braces**: `{job="a", instance=~"$instance"}` — we remove the whole selector; might leave `{job="a"}` — good
- **Variable in regex**: `job=~"$job.*"` — we remove; `job=~"prefix-$job"` — we remove; both correct
- **Empty result**: Removing all selectors can leave `metric{}` or `metric` — we have `emptyBraces` to clean `{}`

**Potential improvement**: More comprehensive regex for variable patterns:
- `\b\w+=~?["']?(?:\{[^}]*\}|[^"']*\$[^"']*|.*)["']?` — complex; current approach is pragmatic

---

## 6. PromQL Function Coverage

**Current** `uppercaseFunc` regex includes 30+ functions. Prometheus has more.

**Add if seen in the wild**:
- `DAY_OF_MONTH`, `DAY_OF_WEEK`, `HOUR`, `MINUTE`, `MONTH`, `YEAR` (rare in dashboards)
- `RAD`, `DEG` (trig)
- `SIN`, `COS`, etc. (uncommon in observability)
- `VECTOR` (already lowercase)

**Recommendation**: Keep current set; add on demand when we hit parse errors. The PromQL parser will fail on unknown functions — we don't need to pre-lowercase everything.

---

## 7. Parse Error Resilience

**Current**: If PromQL parse fails, we still include the panel but mark it with `ParseError`. Good.

**Improvements**:
- **Retry with sanitization**: Try parsing raw expr; if fail, sanitize, parse again; if still fail, mark error
- **Log which variable caused failure**: e.g. "unknown variable $__bucket" — helps users fix dashboards
- **Strict mode flag**: `--strict` that errors on any parse failure instead of including broken panels

---

## 8. Schema Validation (Optional)

**Use Grafana's schema** for validation:
- Not for parsing — for *validating* that we understand the structure
- Could fetch [JSON schema](https://grafana.com/docs/grafana/latest/as-code/observability-as-code/schema-v2/) and validate dashboard structure before parsing
- Catches future Grafana changes early

---

## 9. Implementation Priority

| Priority | Item | Effort | Impact |
|----------|------|--------|--------|
| ~~P0~~ Done | Add `$__range`, `$__range_s` replacement (configurable interval) | Low | Fixes more dashboards |
| **P0** | Datasource filtering (skip non-Prometheus) | Medium | Cleaner output, fewer false panels |
| **P1** | Defensive nil checks for collapsed rows | Low | Avoid rare panics |
| **P1** | Log skipped panels (datasource, empty expr) | Low | Transparency |
| **P2** | Schema v2 / elements layout | High | Future Grafana compatibility |
| **P2** | Preserve `legendFormat` if STS supports it | Medium | Better panel labels |
| **P3** | Schema validation step | Medium | Quality gate |

---

## 10. Test Coverage Ideas

- **Variable matrix**: Dashboards with `$__range`, `$__range_s`, `$resolution`, custom `$interval`
- **Datasource mix**: Dashboard with Prometheus + Loki panels — only Prometheus should appear
- **Schema versions**: Test with `schemaVersion` 1, 16, 38, 39 (Grafana 7–10)
- **Edge titles**: Empty title, unicode, very long
- **Duplicate exprs**: Same expr in multiple targets — we dedupe; verify
- **Malformed JSON**: Partial/corrupt export — graceful error, no crash

---

## Summary

The current parser already handles the majority of real-world dashboards (47 tested, 1,551 panels, zero crashes). The highest-impact, lowest-effort improvements are:

1. **Add `$__range` and `$__range_s`** to sanitization (5 lines)
2. **Filter by datasource** so we only convert Prometheus panels (clearer UX)
3. **Document edge cases** in DASHBOARD_PATTERNS.md as we discover them

Schema v2 and more exotic panel types can wait until users report specific failures.
