<script setup>
import { ref } from 'vue'

const props = defineProps(['parseResult', 'config'])
const emit = defineEmits(['configured', 'back'])

const stsUrl = ref(props.config.stsUrl || '')
const stsToken = ref(props.config.stsToken || '')
const name = ref(props.config.name || props.parseResult?.title || '')
const metricPrefix = ref(props.config.metricPrefix || '')
const interval = ref(props.config.interval || '5m')
const rewriteMetrics = ref(props.config.rewriteMetrics ?? true)
const includeMissing = ref(props.config.includeMissing ?? false)
const showToken = ref(false)

function proceed() {
  emit('configured', {
    stsUrl: stsUrl.value.replace(/\/+$/, ''),
    stsToken: stsToken.value,
    name: name.value,
    metricPrefix: metricPrefix.value,
    interval: interval.value || '5m',
    rewriteMetrics: rewriteMetrics.value,
    includeMissing: includeMissing.value,
  })
}

const panelCount = props.parseResult?.panels?.length || 0
const warnings = props.parseResult?.panels?.filter(p => p.warning)?.length || 0
const metricCount = new Set(props.parseResult?.panels?.flatMap(p => p.metrics || []) || []).size
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-2">Configure Conversion</h2>
      <p class="text-slate-400 leading-relaxed">
        Optionally connect to your SUSE Observability instance to verify which metrics exist. This enables prefix auto-detection and metric availability reports.
      </p>
    </div>

    <!-- Parse summary -->
    <div class="bg-slate-800/30 rounded-2xl ring-1 ring-slate-700/40 p-5">
      <div class="flex items-center gap-4">
        <div class="w-11 h-11 rounded-xl bg-emerald-500/15 flex items-center justify-center flex-shrink-0">
          <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
        </div>
        <div class="flex-1 min-w-0">
          <p class="font-semibold text-slate-200 truncate">{{ name || 'Untitled Dashboard' }}</p>
          <div class="flex items-center gap-3 mt-0.5">
            <span class="text-sm text-slate-400">{{ panelCount }} panel{{ panelCount !== 1 ? 's' : '' }}</span>
            <span class="w-1 h-1 rounded-full bg-slate-700"></span>
            <span class="text-sm text-slate-400">{{ metricCount }} unique metric{{ metricCount !== 1 ? 's' : '' }}</span>
            <span v-if="warnings" class="text-sm text-amber-400">{{ warnings }} warning{{ warnings !== 1 ? 's' : '' }}</span>
          </div>
        </div>
      </div>
    </div>

    <!-- STS connection -->
    <div class="bg-slate-800/20 rounded-2xl ring-1 ring-slate-700/40 p-5 space-y-4">
      <div class="flex items-center gap-3">
        <div class="w-9 h-9 rounded-lg bg-teal-500/15 flex items-center justify-center">
          <svg class="w-5 h-5 text-teal-400" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8zm-1-13h2v6h-2zm0 8h2v2h-2z"/></svg>
        </div>
        <div>
          <h3 class="text-sm font-semibold text-slate-200">SUSE Observability Connection</h3>
          <p class="text-xs text-slate-500">Optional — enables live metric validation and auto-prefix detection</p>
        </div>
      </div>

      <div class="space-y-3 pl-12">
        <div>
          <label class="block text-xs font-medium text-slate-400 mb-1.5">Instance URL</label>
          <input
            v-model="stsUrl"
            type="url"
            placeholder="https://observability.example.com"
            class="w-full bg-slate-900/80 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-teal-500/40 focus:border-teal-500/40 transition"
          />
        </div>
        <div>
          <label class="block text-xs font-medium text-slate-400 mb-1.5">API Token</label>
          <div class="relative">
            <input
              v-model="stsToken"
              :type="showToken ? 'text' : 'password'"
              placeholder="Paste your API token here"
              class="w-full bg-slate-900/80 border border-slate-700/80 rounded-xl px-4 py-2.5 pr-12 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-teal-500/40 focus:border-teal-500/40 transition"
            />
            <button @click="showToken = !showToken" class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-600 hover:text-slate-300 transition p-0.5">
              <svg v-if="!showToken" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg>
              <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" /></svg>
            </button>
          </div>
          <p class="text-[10px] text-slate-600 mt-1.5">Find your token in SUSE Observability &rarr; Settings &rarr; API Tokens</p>
        </div>
      </div>
    </div>

    <!-- Conversion options -->
    <div class="bg-slate-800/20 rounded-2xl ring-1 ring-slate-700/40 p-5 space-y-4">
      <h3 class="text-sm font-semibold text-slate-200">Output Options</h3>
      <div class="space-y-4">
        <div>
          <label class="block text-xs font-medium text-slate-400 mb-1.5">Dashboard Name</label>
          <input
            v-model="name"
            type="text"
            placeholder="My Dashboard"
            class="w-full bg-slate-900/80 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition"
          />
          <p class="text-[10px] text-slate-600 mt-1">This will be the dashboard name in SUSE Observability</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-slate-400 mb-1.5">PromQL Interval</label>
          <select
            v-model="interval"
            class="w-full bg-slate-900/80 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-slate-200 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition"
          >
            <option value="1m">1m</option>
            <option value="5m">5m (recommended)</option>
            <option value="15m">15m</option>
            <option value="1h">1h</option>
          </select>
          <p class="text-[10px] text-slate-600 mt-1">Replaces Grafana variables like <code class="text-slate-500">$__rate_interval</code>, <code class="text-slate-500">$__interval</code>. Use 5m for typical scrape intervals (15s–1m).</p>
        </div>
        <div>
          <label class="block text-xs font-medium text-slate-400 mb-1.5">Metric Prefix Override</label>
          <input
            v-model="metricPrefix"
            type="text"
            placeholder="Auto-detected (e.g. postgresql, mysql)"
            class="w-full bg-slate-900/80 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition"
          />
          <p class="text-[10px] text-slate-600 mt-1">The STS agent often adds a prefix (e.g., <code class="text-slate-500">pg_up</code> becomes <code class="text-slate-500">postgresql_pg_up</code>). Leave empty to auto-detect.</p>
        </div>
        <div class="flex flex-col gap-3">
          <label class="flex items-center gap-3 cursor-pointer" @click="rewriteMetrics = !rewriteMetrics">
            <div :class="['w-10 h-5.5 rounded-full relative transition-colors duration-200', rewriteMetrics ? 'bg-emerald-500' : 'bg-slate-700']">
              <div :class="['w-4 h-4 rounded-full bg-white absolute top-[3px] transition-transform duration-200 shadow-sm', rewriteMetrics ? 'translate-x-[22px]' : 'translate-x-[3px]']" />
            </div>
            <div>
              <span class="text-sm text-slate-300">Rewrite metric names with detected prefix</span>
              <p class="text-[10px] text-slate-600">Automatically adjusts metric names to match STS naming</p>
            </div>
          </label>
          <label class="flex items-center gap-3 cursor-pointer" @click="includeMissing = !includeMissing">
            <div :class="['w-10 h-5.5 rounded-full relative transition-colors duration-200', includeMissing ? 'bg-emerald-500' : 'bg-slate-700']">
              <div :class="['w-4 h-4 rounded-full bg-white absolute top-[3px] transition-transform duration-200 shadow-sm', includeMissing ? 'translate-x-[22px]' : 'translate-x-[3px]']" />
            </div>
            <div>
              <span class="text-sm text-slate-300">Include panels with missing metrics</span>
              <p class="text-[10px] text-slate-600">Keep all panels even if their metrics aren't found in STS</p>
            </div>
          </label>
        </div>
      </div>
    </div>

    <!-- Actions -->
    <div class="flex gap-3">
      <button @click="$emit('back')" class="px-5 py-3 rounded-xl text-sm font-medium text-slate-400 hover:text-white bg-slate-800/60 hover:bg-slate-700/60 ring-1 ring-slate-700/40 transition-all">
        Back
      </button>
      <button @click="proceed" class="flex-1 py-3 rounded-xl font-semibold text-sm bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25 transition-all">
        {{ stsUrl && stsToken ? 'Check Metrics & Continue' : 'Skip Check & Generate YAML' }}
      </button>
    </div>
  </div>
</template>
