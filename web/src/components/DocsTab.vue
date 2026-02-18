<script setup>
import { ref } from 'vue'

const activeSection = ref('overview')

const sections = [
  { id: 'overview', label: 'Overview' },
  { id: 'how-it-works', label: 'How It Works' },
  { id: 'sanitization', label: 'PromQL Sanitization' },
  { id: 'metrics', label: 'Metric Extraction' },
  { id: 'prefix', label: 'Prefix Detection' },
  { id: 'parsing', label: 'Grafana Parsing' },
  { id: 'yaml', label: 'STS YAML Format' },
  { id: 'cli', label: 'CLI Reference' },
]

const sanitizationRules = [
  { name: 'Rate interval', before: 'rate(http_requests_total[$__rate_interval])', after: 'rate(http_requests_total[5m])' },
  { name: 'Interval', before: 'increase(errors_total[${__interval}])', after: 'increase(errors_total[5m])' },
  { name: 'Range variable', before: 'avg_over_time(cpu_usage[$interval])', after: 'avg_over_time(cpu_usage[5m])' },
  { name: '$__range_s', before: 'increase(errors[$__range_s])', after: 'increase(errors[300])' },
  { name: 'Variable bake-in', before: 'up{namespace="$namespace"}', after: 'up{namespace="prod"} (with -v namespace=prod)' },
  { name: 'Variable strip (no override)', before: 'up{job=~"$job", namespace="$namespace"}', after: 'up' },
  { name: 'Mixed labels', before: 'rate(http_total{method="GET", job=~"$job"}[5m])', after: 'rate(http_total{method="GET"}[5m])' },
  { name: 'Dangling comma', before: 'node_memory{, instance="a"}', after: 'node_memory{instance="a"}' },
  { name: 'Trailing comma', before: 'node_cpu{mode="idle",}', after: 'node_cpu{mode="idle"}' },
  { name: 'Empty braces', before: 'node_load1{}', after: 'node_load1' },
  { name: 'Uppercase functions', before: 'SUM(RATE(requests_total[5m]))', after: 'sum(rate(requests_total[5m]))' },
]

const metricExamples = [
  { expr: 'rate(http_requests_total[5m])', metrics: ['http_requests_total'], note: 'Simple rate wrapper' },
  { expr: 'histogram_quantile(0.99, sum(rate(http_duration_seconds_bucket[5m])) by (le))', metrics: ['http_duration_seconds_bucket'], note: 'Deeply nested histogram' },
  { expr: 'node_memory_MemTotal_bytes - node_memory_MemAvailable_bytes', metrics: ['node_memory_MemTotal_bytes', 'node_memory_MemAvailable_bytes'], note: 'Binary expression with two metrics' },
  { expr: 'sum by (pod) (rate(container_cpu_usage_seconds_total[5m])) / sum by (pod) (kube_pod_container_resource_limits)', metrics: ['container_cpu_usage_seconds_total', 'kube_pod_container_resource_limits'], note: 'Division of aggregations' },
  { expr: 'count(up == 1)', metrics: ['up'], note: 'Comparison operator' },
]

const prefixExamples = [
  { from: 'pg_up', to: 'postgresql_pg_up', how: 'Prefix added' },
  { from: 'pg_stat_activity_count', to: 'postgresql_pg_stat_activity_count', how: 'Prefix added' },
  { from: 'pg_locks_count', to: 'postgresql_pg_locks_count', how: 'Prefix added' },
  { from: 'pg_stat_bgwriter_buffers_alloc_total', to: 'postgresql_pg_stat_bgwriter_buffers_alloc', how: 'Prefix + _total stripped' },
]

const parsingFeatures = [
  { icon: '\u{1F4E6}', title: 'Standard panels', desc: 'panels[].targets[].expr \u2014 the most common pattern in Grafana 7+.' },
  { icon: '\u{1F4C2}', title: 'Row panels (type: "row")', desc: 'Grafana groups panels into collapsible rows. Nested panels inside rows are recursively extracted.' },
  { icon: '\u{1F4DC}', title: 'Legacy rows[]', desc: 'Older Grafana versions use a top-level rows[] array instead of putting rows inside panels[]. Both are supported.' },
  { icon: '\u{1F504}', title: 'Deeply nested panels', desc: 'Some dashboards nest panels inside other panels (not just rows). The parser recursively walks all children.' },
  { icon: '\u{1F195}', title: 'Grafana 8+ options.queries[]', desc: 'Newer panels may put queries in options.queries[] instead of targets[]. Both locations are checked.' },
  { icon: '\u{1F3F7}', title: 'targets[].query fallback', desc: 'Some data sources use "query" instead of "expr". Odyssey checks both fields.' },
  { icon: '\u{1F4DD}', title: 'Title inheritance', desc: 'If a panel has no title, it inherits from the parent row. If neither has a title, a fallback "Panel N" name is generated.' },
]

const pipelineSteps = [
  { color: 'orange', title: 'Parse', desc: 'Grafana JSON is parsed recursively. Every PromQL expression is extracted from panels, nested rows, collapsed panels, and Grafana 8+ options.queries[]. Legacy rows[] are also supported.' },
  { color: 'emerald', title: 'Sanitise', desc: 'Built-in time variables ($__rate_interval, $__interval, $__range_s) are replaced with your chosen interval (default 5m). Template variables ($namespace, $job, etc.) are baked in using Grafana defaults or your overrides — any remaining unknowns are stripped. Function names are lowercased.' },
  { color: 'blue', title: 'Validate', desc: 'Extracted metric names are checked against the SUSE Observability Prometheus API. Prefix detection identifies if your metrics have a namespace prefix (e.g. postgresql_ or mysql_). Counter _total suffixes are handled transparently.' },
  { color: 'teal', title: 'Generate', desc: 'A valid STS dashboard YAML is produced with a Grid layout, TimeSeriesChart panels, PrometheusTimeSeriesQuery queries, and proper panel IDs. The YAML can be applied directly or downloaded.' },
]

const stepColors = {
  orange: { bg: 'rgba(249,115,22,0.15)', fg: '#fb923c' },
  emerald: { bg: 'rgba(16,185,129,0.15)', fg: '#34d399' },
  blue: { bg: 'rgba(59,130,246,0.15)', fg: '#60a5fa' },
  teal: { bg: 'rgba(20,184,166,0.15)', fg: '#2dd4bf' },
}

const cliCommands = [
  {
    name: 'odyssey',
    desc: 'Interactive wizard \u2014 walks you through upload, config, check, and export step by step. Supports both local file upload and pulling directly from Grafana.',
    usage: 'odyssey',
    flags: [],
  },
  {
    name: 'odyssey convert',
    desc: 'Non-interactive dashboard conversion. Parses Grafana JSON, checks metrics, and generates STS YAML.',
    usage: 'odyssey convert -i dashboard.json --sts-url https://obs.example.com --sts-token TOKEN',
    flags: [
      { flag: '-i, --input', desc: 'Grafana JSON file(s) \u2014 supports globs (e.g. *.json)' },
      { flag: '--sts-url', desc: 'SUSE Observability base URL' },
      { flag: '--sts-token', desc: 'SUSE Observability API token' },
      { flag: '-o, --output', desc: 'Output YAML file path (default: <input>.sts.yaml)' },
      { flag: '--name', desc: 'Dashboard name in STS' },
      { flag: '--interval', desc: 'PromQL interval for rate/irate (default: 5m; e.g. 1m, 15m, 1h)' },
      { flag: '-v, --variable', desc: 'Bake variable value into queries (repeatable, e.g. -v namespace=prod)' },
      { flag: '--include-missing', desc: 'Include panels whose metrics are not in STS' },
      { flag: '--no-rewrite', desc: "Don't rewrite metric names with detected prefix" },
    ],
  },
  {
    name: 'odyssey pull',
    desc: 'Connect to a Grafana instance and list or download dashboards.',
    usage: 'odyssey pull --grafana-url https://grafana.example.com --grafana-token TOKEN',
    flags: [
      { flag: '--grafana-url', desc: 'Grafana base URL' },
      { flag: '--grafana-token', desc: 'Grafana API key or service account token' },
      { flag: '--uid', desc: 'Dashboard UID to fetch (omit to list all)' },
      { flag: '-o, --output-dir', desc: 'Directory to save downloaded JSON (default: .)' },
    ],
  },
  {
    name: 'odyssey apply',
    desc: 'Apply a generated STS YAML file using the sts CLI. Requires sts to be installed and configured.',
    usage: 'odyssey apply dashboard.sts.yaml',
    flags: [],
  },
  {
    name: 'odyssey check',
    desc: 'Check metric availability without generating YAML. Useful for a quick audit.',
    usage: 'odyssey check -i dashboard.json --sts-url URL --sts-token TOKEN',
    flags: [
      { flag: '--sts-url', desc: 'SUSE Observability base URL' },
      { flag: '--sts-token', desc: 'SUSE Observability API token' },
      { flag: '--interval', desc: 'PromQL interval for rate/irate (default: 5m)' },
      { flag: '-v, --variable', desc: 'Bake variable value into queries (repeatable, e.g. -v namespace=prod)' },
    ],
  },
  {
    name: 'odyssey server',
    desc: 'Start the web UI on a local port.',
    usage: 'odyssey server --port 8080',
    flags: [
      { flag: '-p, --port', desc: 'Port to listen on (default: 8080)' },
    ],
  },
]

const fullExampleBefore = 'SUM(RATE(nginx_http_requests_total{job=~"$job", namespace="$namespace"}[$__rate_interval])) BY (status)'
const fullExampleAfter = 'sum(rate(nginx_http_requests_total[5m])) BY (status)'

function scrollTo(id) {
  activeSection.value = id
  document.getElementById('doc-' + id)?.scrollIntoView({ behavior: 'smooth', block: 'start' })
}
</script>

<template>
  <div class="max-w-7xl mx-auto px-6 py-8">
    <div class="grid grid-cols-1 lg:grid-cols-4 gap-8">

      <!-- Sidebar navigation -->
      <nav class="hidden lg:block">
        <div class="sticky top-24 space-y-1">
          <h3 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3 px-3">Documentation</h3>
          <button
            v-for="s in sections"
            :key="s.id"
            @click="scrollTo(s.id)"
            :class="[
              'block w-full text-left text-sm px-3 py-2 rounded-lg transition-all',
              activeSection === s.id
                ? 'bg-emerald-500/10 text-emerald-400 font-medium'
                : 'text-slate-500 hover:text-slate-300 hover:bg-slate-800/40',
            ]"
          >{{ s.label }}</button>
        </div>
      </nav>

      <!-- Content -->
      <div class="lg:col-span-3 space-y-16">

        <!-- Overview -->
        <section id="doc-overview">
          <h2 class="text-2xl font-bold text-white mb-4">What is Odyssey?</h2>
          <div class="prose-custom">
            <p>
              Odyssey is a migration tool that converts <strong>Grafana dashboards</strong> into
              <strong>SUSE Observability</strong> (formerly StackState) dashboard YAML. It handles the entire pipeline:
              parsing Grafana JSON, sanitising PromQL queries, validating metrics against a live instance,
              and generating production-ready YAML that can be applied with <code>sts dashboard apply</code>.
            </p>
            <p>
              You can use it three ways: this <strong>web UI</strong>, the <strong>interactive CLI wizard</strong>
              (<code>odyssey</code> with no arguments), or <strong>non-interactive flags</strong>
              (<code>odyssey convert</code>).
            </p>
            <div class="mt-6 bg-slate-800/30 rounded-2xl ring-1 ring-slate-700/40 p-5 grid sm:grid-cols-3 gap-4 text-center not-prose">
              <div>
                <div class="text-3xl font-bold text-emerald-400">47</div>
                <div class="text-xs text-slate-500 mt-1">Dashboards Tested</div>
              </div>
              <div>
                <div class="text-3xl font-bold text-emerald-400">1,551</div>
                <div class="text-xs text-slate-500 mt-1">Panels Parsed</div>
              </div>
              <div>
                <div class="text-3xl font-bold text-emerald-400">0</div>
                <div class="text-xs text-slate-500 mt-1">Parser Crashes</div>
              </div>
            </div>
          </div>
        </section>

        <!-- How It Works -->
        <section id="doc-how-it-works">
          <h2 class="text-2xl font-bold text-white mb-4">How the Migration Works</h2>
          <div class="prose-custom">
            <p>The migration pipeline has four stages. Each stage is visible as a wizard step in the UI.</p>
          </div>
          <div class="mt-6 space-y-4 not-prose">
            <div v-for="(item, i) in pipelineSteps" :key="i" class="flex gap-4 items-start">
              <div class="w-10 h-10 rounded-xl flex items-center justify-center flex-shrink-0 font-bold text-sm"
              :style="{
                backgroundColor: stepColors[item.color].bg,
                color: stepColors[item.color].fg,
              }">{{ i + 1 }}</div>
              <div>
                <h4 class="font-semibold text-slate-200">{{ item.title }}</h4>
                <p class="text-sm text-slate-500 leading-relaxed mt-1">{{ item.desc }}</p>
              </div>
            </div>
          </div>
        </section>

        <!-- PromQL Sanitization -->
        <section id="doc-sanitization">
          <h2 class="text-2xl font-bold text-white mb-4">PromQL Sanitization</h2>
          <div class="prose-custom">
            <p>
              Grafana dashboards use template variables (<code>$__rate_interval</code>, <code>$__interval</code>,
              <code>$__range_s</code>, <code>$job</code>, <code>$namespace</code>) that don't exist in SUSE Observability.
              Odyssey applies several transformations to make the PromQL valid.
            </p>
            <p>
              <strong>Configurable interval</strong> — You can choose the replacement value for range variables.
              In the web UI, use the "PromQL Interval" dropdown in Configure (1m, 5m, 15m, 1h). In the CLI,
              use <code>--interval 5m</code>. Default is <code>5m</code>, which suits typical Prometheus scrape
              intervals (15s–1m).
            </p>
            <p>
              <strong>Variable bake-in</strong> — Odyssey parses Grafana's <code>templating.list</code> to extract
              default values for template variables (e.g. <code>$namespace</code>, <code>$job</code>). These defaults
              are baked into queries, turning <code>namespace="$namespace"</code> into <code>namespace="prod"</code>.
              You can override any value via <code>--variable namespace=prod</code> (CLI) or the Variable Overrides
              section in the Configure step (web UI). Variables without a known value are stripped as before.
            </p>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Transformation Rules</h3>
          <div class="not-prose overflow-hidden rounded-xl ring-1 ring-slate-700/40">
            <table class="w-full text-sm">
              <thead class="bg-slate-800/50">
                <tr>
                  <th class="text-left px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">Rule</th>
                  <th class="text-left px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">Before</th>
                  <th class="text-left px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">After</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                <tr v-for="(rule, i) in sanitizationRules" :key="i" class="hover:bg-slate-800/20 transition-colors">
                  <td class="px-4 py-2.5 text-slate-300 font-medium whitespace-nowrap">{{ rule.name }}</td>
                  <td class="px-4 py-2.5"><code class="text-red-400/80 text-xs bg-red-500/5 px-1.5 py-0.5 rounded">{{ rule.before }}</code></td>
                  <td class="px-4 py-2.5"><code class="text-emerald-400/80 text-xs bg-emerald-500/5 px-1.5 py-0.5 rounded">{{ rule.after }}</code></td>
                </tr>
              </tbody>
            </table>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Supported Variables</h3>
          <div class="prose-custom">
            <p>Grafana template variables in <strong>label selectors</strong> (e.g. <code>$job</code>, <code>$namespace</code>)
              are handled in two ways: if a default value is found in <code>templating.list</code> or provided via
              <code>--variable</code>, the value is <strong>baked in</strong>. Otherwise, the selector is <strong>removed</strong>.
              Variables inside <strong>range brackets</strong> (<code>[$var]</code>) are replaced with your chosen
              interval (default <code>5m</code>). <code>$__range_s</code> is replaced with the interval in seconds
              (e.g. 300 for 5m, 60 for 1m).</p>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Full Example</h3>
          <div class="not-prose space-y-3">
            <div class="bg-slate-800/30 rounded-xl ring-1 ring-slate-700/40 p-4">
              <div class="text-[10px] text-slate-500 uppercase tracking-wider font-semibold mb-2">Input (Grafana)</div>
              <code class="text-sm text-red-400/80 leading-relaxed block">{{ fullExampleBefore }}</code>
            </div>
            <div class="flex justify-center">
              <svg class="w-5 h-5 text-slate-600" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 14l-7 7m0 0l-7-7m7 7V3"/></svg>
            </div>
            <div class="bg-slate-800/30 rounded-xl ring-1 ring-emerald-500/20 p-4">
              <div class="text-[10px] text-emerald-500 uppercase tracking-wider font-semibold mb-2">Output (Sanitised)</div>
              <code class="text-sm text-emerald-400/80 leading-relaxed block">{{ fullExampleAfter }}</code>
            </div>
          </div>
          <div class="prose-custom mt-3">
            <p class="text-xs text-slate-600">
              Steps applied: (1) SUM/RATE lowercased,
              (2) $__rate_interval replaced with 5m,
              (3) known variable values baked into label selectors; unknowns removed,
              (4) empty braces cleaned.
            </p>
          </div>
        </section>

        <!-- Metric Extraction -->
        <section id="doc-metrics">
          <h2 class="text-2xl font-bold text-white mb-4">Metric Extraction</h2>
          <div class="prose-custom">
            <p>
              After sanitisation, Odyssey uses the <strong>official Prometheus PromQL parser</strong>
              (<code>github.com/prometheus/prometheus/promql/parser</code>) to build an Abstract Syntax Tree (AST)
              of each expression. It then walks the tree looking for <code>VectorSelector</code> nodes
              — these are the actual metric references.
            </p>
            <p>
              This is far more reliable than regex-based extraction because it correctly handles:
            </p>
          </div>
          <div class="mt-4 not-prose space-y-3">
            <div v-for="(ex, i) in metricExamples" :key="i" class="bg-slate-800/30 rounded-xl ring-1 ring-slate-700/40 p-4">
              <div class="flex items-start justify-between gap-4 mb-2">
                <code class="text-xs text-slate-300 leading-relaxed">{{ ex.expr }}</code>
                <span class="text-[10px] text-slate-600 whitespace-nowrap flex-shrink-0">{{ ex.note }}</span>
              </div>
              <div class="flex flex-wrap gap-1.5">
                <span v-for="m in ex.metrics" :key="m" class="inline-flex items-center px-2 py-0.5 rounded-md text-xs font-mono bg-emerald-500/10 text-emerald-400 ring-1 ring-emerald-500/20">{{ m }}</span>
              </div>
            </div>
          </div>
          <div class="prose-custom mt-4">
            <p>If the PromQL parser fails (e.g. because the expression still contains unsanitised Grafana
              syntax), Odyssey falls back gracefully — the panel is still included but marked as having
              a parse error. No crash, no data loss.</p>
          </div>
        </section>

        <!-- Prefix Detection -->
        <section id="doc-prefix">
          <h2 class="text-2xl font-bold text-white mb-4">Metric Prefix Detection &amp; Rewriting</h2>
          <div class="prose-custom">
            <p>
              When you deploy an exporter (e.g. PostgreSQL, MySQL, NGINX) with the SUSE Observability agent,
              the openmetrics integration often prepends a <strong>namespace prefix</strong> to every metric.
              For example, a Grafana dashboard might reference <code>pg_stat_activity_count</code>, but in STS
              the metric is stored as <code>postgresql_pg_stat_activity_count</code>.
            </p>
            <p>
              Odyssey handles this automatically using a <strong>suffix index</strong>.
            </p>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">How It Works</h3>
          <div class="not-prose space-y-4 mt-4">
            <div class="bg-slate-800/30 rounded-xl ring-1 ring-slate-700/40 p-4 space-y-3">
              <div class="flex gap-3 items-start">
                <div class="w-7 h-7 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0 text-xs font-bold text-blue-400">1</div>
                <div>
                  <h4 class="text-sm font-semibold text-slate-200">Build a Suffix Index</h4>
                  <p class="text-xs text-slate-500 mt-1">
                    All metric names from STS are split at every <code>_</code> boundary. Each suffix is stored
                    in a reverse lookup map. So <code>postgresql_pg_stat_activity_count</code> creates entries
                    for <code>pg_stat_activity_count</code>, <code>stat_activity_count</code>, <code>activity_count</code>, and <code>count</code>.
                  </p>
                </div>
              </div>
              <div class="flex gap-3 items-start">
                <div class="w-7 h-7 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0 text-xs font-bold text-blue-400">2</div>
                <div>
                  <h4 class="text-sm font-semibold text-slate-200">Match Grafana Metric Names</h4>
                  <p class="text-xs text-slate-500 mt-1">
                    When checking <code>pg_stat_activity_count</code> from the Grafana dashboard, Odyssey first
                    tries an exact match. If that fails, it checks the suffix index and finds
                    <code>postgresql_pg_stat_activity_count</code>.
                  </p>
                </div>
              </div>
              <div class="flex gap-3 items-start">
                <div class="w-7 h-7 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0 text-xs font-bold text-blue-400">3</div>
                <div>
                  <h4 class="text-sm font-semibold text-slate-200">Auto-detect the Prefix</h4>
                  <p class="text-xs text-slate-500 mt-1">
                    If &ge;2 metrics share the same prefix (e.g. <code>postgresql_</code>), it's automatically
                    detected and offered for rewriting. The most common prefix wins.
                  </p>
                </div>
              </div>
              <div class="flex gap-3 items-start">
                <div class="w-7 h-7 rounded-lg bg-blue-500/15 flex items-center justify-center flex-shrink-0 text-xs font-bold text-blue-400">4</div>
                <div>
                  <h4 class="text-sm font-semibold text-slate-200">Rewrite Expressions</h4>
                  <p class="text-xs text-slate-500 mt-1">
                    Every bare metric name in the PromQL is replaced with its prefixed counterpart.
                    The expressions become valid against STS without manual editing.
                  </p>
                </div>
              </div>
            </div>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Example: PostgreSQL Dashboard</h3>
          <div class="not-prose overflow-hidden rounded-xl ring-1 ring-slate-700/40">
            <table class="w-full text-sm">
              <thead class="bg-slate-800/50">
                <tr>
                  <th class="text-left px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">Grafana Metric</th>
                  <th class="text-left px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">STS Metric</th>
                  <th class="text-left px-4 py-2.5 text-xs font-semibold text-slate-400 uppercase tracking-wider">How</th>
                </tr>
              </thead>
              <tbody class="divide-y divide-slate-800/60">
                <tr v-for="row in prefixExamples" :key="row.from" class="hover:bg-slate-800/20 transition-colors">
                  <td class="px-4 py-2.5"><code class="text-xs text-red-400/80">{{ row.from }}</code></td>
                  <td class="px-4 py-2.5"><code class="text-xs text-emerald-400/80">{{ row.to }}</code></td>
                  <td class="px-4 py-2.5 text-xs text-slate-500">{{ row.how }}</td>
                </tr>
              </tbody>
            </table>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Counter _total Suffix</h3>
          <div class="prose-custom">
            <p>
              Prometheus counters conventionally end in <code>_total</code> (e.g.
              <code>http_requests_total</code>). However, the SUSE Observability openmetrics agent
              strips this suffix, storing the metric as <code>http_requests</code>. Odyssey
              automatically tries matching without <code>_total</code> when the exact name doesn't exist.
            </p>
          </div>
        </section>

        <!-- Grafana Parsing -->
        <section id="doc-parsing">
          <h2 class="text-2xl font-bold text-white mb-4">Grafana JSON Parsing</h2>
          <div class="prose-custom">
            <p>
              Grafana dashboards have varied and sometimes deeply nested JSON structures across versions.
              Odyssey's parser handles all common patterns:
            </p>
          </div>
          <div class="mt-4 not-prose space-y-3">
            <div v-for="(feat, i) in parsingFeatures" :key="i" class="flex gap-3 items-start bg-slate-800/20 rounded-xl ring-1 ring-slate-700/30 px-4 py-3">
              <span class="text-lg flex-shrink-0">{{ feat.icon }}</span>
              <div>
                <h4 class="text-sm font-semibold text-slate-200">{{ feat.title }}</h4>
                <p class="text-xs text-slate-500 mt-0.5 leading-relaxed">{{ feat.desc }}</p>
              </div>
            </div>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Example JSON Structure</h3>
          <div class="not-prose bg-slate-800/30 rounded-xl ring-1 ring-slate-700/40 p-4 overflow-x-auto">
<pre class="text-xs text-slate-400 font-mono leading-relaxed"><span class="text-slate-600">// Grafana dashboard.json (simplified)</span>
{
  <span class="text-emerald-400">"title"</span>: <span class="text-amber-400">"My Dashboard"</span>,
  <span class="text-emerald-400">"panels"</span>: [
    {
      <span class="text-emerald-400">"type"</span>: <span class="text-amber-400">"row"</span>,
      <span class="text-emerald-400">"title"</span>: <span class="text-amber-400">"HTTP Metrics"</span>,
      <span class="text-emerald-400">"panels"</span>: [
        {
          <span class="text-emerald-400">"title"</span>: <span class="text-amber-400">"Request Rate"</span>,
          <span class="text-emerald-400">"type"</span>: <span class="text-amber-400">"timeseries"</span>,
          <span class="text-emerald-400">"targets"</span>: [
            { <span class="text-emerald-400">"expr"</span>: <span class="text-amber-400">"rate(http_requests_total[$__rate_interval])"</span> }
          ]
        }
      ]
    },
    {
      <span class="text-emerald-400">"title"</span>: <span class="text-amber-400">"Error Rate"</span>,
      <span class="text-emerald-400">"type"</span>: <span class="text-amber-400">"graph"</span>,
      <span class="text-emerald-400">"targets"</span>: [
        { <span class="text-emerald-400">"expr"</span>: <span class="text-amber-400">"sum(rate(http_errors_total[5m]))"</span> }
      ]
    }
  ],
  <span class="text-emerald-400">"rows"</span>: [
    {
      <span class="text-emerald-400">"title"</span>: <span class="text-amber-400">"Legacy Section"</span>,
      <span class="text-emerald-400">"panels"</span>: [
        { <span class="text-emerald-400">"targets"</span>: [{ <span class="text-emerald-400">"expr"</span>: <span class="text-amber-400">"node_load1"</span> }] }
      ]
    }
  ]
}</pre>
          </div>
          <div class="prose-custom mt-3">
            <p class="text-xs text-slate-600">
              Odyssey would extract 3 panels from this: "Request Rate" (from nested row panel),
              "Error Rate" (from top-level panel), and "Legacy Section" (from legacy rows[]).
            </p>
          </div>
        </section>

        <!-- STS YAML Format -->
        <section id="doc-yaml">
          <h2 class="text-2xl font-bold text-white mb-4">STS YAML Output Format</h2>
          <div class="prose-custom">
            <p>
              The generated YAML conforms to the SUSE Observability <code>DashboardWriteSchema</code>
              and can be applied directly with <code>sts dashboard apply --file dashboard.sts.yaml</code>.
            </p>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Output Structure</h3>
          <div class="not-prose bg-slate-800/30 rounded-xl ring-1 ring-slate-700/40 p-4 overflow-x-auto">
<pre class="text-xs text-slate-400 font-mono leading-relaxed"><span class="text-emerald-400">name</span>: <span class="text-amber-400">My Dashboard</span>
<span class="text-emerald-400">scope</span>: <span class="text-amber-400">publicDashboard</span>
<span class="text-emerald-400">dashboard</span>:
  <span class="text-emerald-400">metadata</span>:
    <span class="text-emerald-400">project</span>: <span class="text-amber-400">'{"saveVariables":false}'</span>
  <span class="text-emerald-400">spec</span>:
    <span class="text-emerald-400">layouts</span>:
      - <span class="text-emerald-400">kind</span>: <span class="text-amber-400">Grid</span>
        <span class="text-emerald-400">spec</span>:
          <span class="text-emerald-400">items</span>:
            - <span class="text-emerald-400">x</span>: <span class="text-blue-400">0</span>
              <span class="text-emerald-400">y</span>: <span class="text-blue-400">0</span>
              <span class="text-emerald-400">width</span>: <span class="text-blue-400">8</span>
              <span class="text-emerald-400">height</span>: <span class="text-blue-400">2</span>
              <span class="text-emerald-400">content</span>:
                <span class="text-emerald-400">$ref</span>: <span class="text-amber-400">"#/spec/panels/a1b2c3d4e5f"</span>
    <span class="text-emerald-400">panels</span>:
      <span class="text-emerald-400">a1b2c3d4e5f</span>:
        <span class="text-emerald-400">spec</span>:
          <span class="text-emerald-400">display</span>:
            <span class="text-emerald-400">name</span>: <span class="text-amber-400">Request Rate</span>
          <span class="text-emerald-400">plugin</span>:
            <span class="text-emerald-400">kind</span>: <span class="text-amber-400">TimeSeriesChart</span>
            <span class="text-emerald-400">spec</span>:
              <span class="text-emerald-400">legend</span>:
                <span class="text-emerald-400">mode</span>: <span class="text-amber-400">list</span>
                <span class="text-emerald-400">position</span>: <span class="text-amber-400">right</span>
                <span class="text-emerald-400">show</span>: <span class="text-blue-400">true</span>
              <span class="text-emerald-400">yAxis</span>:
                <span class="text-emerald-400">show</span>: <span class="text-blue-400">true</span>
          <span class="text-emerald-400">queries</span>:
            - <span class="text-emerald-400">kind</span>: <span class="text-amber-400">TimeSeriesQuery</span>
              <span class="text-emerald-400">spec</span>:
                <span class="text-emerald-400">plugin</span>:
                  <span class="text-emerald-400">kind</span>: <span class="text-amber-400">PrometheusTimeSeriesQuery</span>
                  <span class="text-emerald-400">spec</span>:
                    <span class="text-emerald-400">query</span>: <span class="text-amber-400">rate(http_requests_total[5m])</span></pre>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Layout Grid</h3>
          <div class="prose-custom">
            <p>Panels are arranged in a <strong>3-column grid</strong>. Each cell is 8 units wide and 2 units
              tall. Panels fill left-to-right, top-to-bottom. The grid coordinates are:</p>
          </div>
          <div class="not-prose mt-3 grid grid-cols-3 gap-2">
            <div v-for="n in 6" :key="n" class="bg-slate-800/40 rounded-lg ring-1 ring-slate-700/30 p-3 text-center">
              <div class="text-xs text-slate-300 font-mono">Panel {{ n }}</div>
              <div class="text-[10px] text-slate-600 mt-0.5">x={{ ((n-1) % 3) * 8 }}, y={{ Math.floor((n-1) / 3) * 2 }}</div>
            </div>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">Panel IDs</h3>
          <div class="prose-custom">
            <p>Each panel gets a unique 11-character hex ID derived from a SHA-256 hash of the PromQL
              expression and panel index. This ensures stable, deterministic IDs across re-runs.</p>
          </div>
        </section>

        <!-- CLI Reference -->
        <section id="doc-cli">
          <h2 class="text-2xl font-bold text-white mb-4">CLI Reference</h2>
          <div class="prose-custom">
            <p>Odyssey provides several commands. Run without arguments for the interactive wizard.</p>
          </div>

          <div class="mt-6 not-prose space-y-6">
            <div v-for="cmd in cliCommands" :key="cmd.name" class="bg-slate-800/30 rounded-2xl ring-1 ring-slate-700/40 p-5 space-y-3">
              <div class="flex items-center gap-3">
                <code class="text-sm font-bold text-emerald-400">{{ cmd.name }}</code>
              </div>
              <p class="text-sm text-slate-400 leading-relaxed">{{ cmd.desc }}</p>
              <div class="bg-slate-900/60 rounded-lg px-3 py-2">
                <code class="text-xs text-slate-300">$ {{ cmd.usage }}</code>
              </div>
              <div v-if="cmd.flags.length > 0" class="space-y-1.5">
                <div v-for="f in cmd.flags" :key="f.flag" class="flex gap-3 text-xs">
                  <code class="text-slate-300 flex-shrink-0 min-w-[10rem]">{{ f.flag }}</code>
                  <span class="text-slate-500">{{ f.desc }}</span>
                </div>
              </div>
            </div>
          </div>

          <h3 class="text-lg font-semibold text-slate-200 mt-8 mb-3">End-to-End Example</h3>
          <div class="not-prose bg-slate-800/30 rounded-xl ring-1 ring-slate-700/40 p-4 space-y-2">
<pre class="text-xs text-slate-400 font-mono leading-relaxed"><span class="text-slate-600"># 1. Pull a dashboard from Grafana</span>
<span class="text-emerald-400">$</span> odyssey pull --grafana-url https://grafana.internal \
    --grafana-token glsa_xxxxx --uid abc123

<span class="text-slate-600"># 2. Convert to STS YAML</span>
<span class="text-emerald-400">$</span> odyssey convert -i abc123.json \
    --sts-url https://obs.example.com \
    --sts-token &lt;token&gt; \
    --name "My NGINX Dashboard"

<span class="text-slate-600"># 3. Apply to SUSE Observability</span>
<span class="text-emerald-400">$</span> odyssey apply abc123.sts.yaml</pre>
          </div>
        </section>

      </div>
    </div>
  </div>
</template>

<style scoped>
.prose-custom p {
  font-size: 0.875rem;
  line-height: 1.625;
  color: rgb(148 163 184);
  margin-bottom: 0.75rem;
}
.prose-custom strong {
  color: rgb(226 232 240);
  font-weight: 600;
}
.prose-custom code {
  font-size: 0.75rem;
  color: rgb(52 211 153 / 0.8);
  background-color: rgb(16 185 129 / 0.05);
  padding: 0.125rem 0.25rem;
  border-radius: 0.25rem;
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
.not-prose code {
  font-family: ui-monospace, SFMono-Regular, Menlo, Monaco, Consolas, monospace;
}
</style>
