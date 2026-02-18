<script setup>
import { ref } from 'vue'
import StepUpload from './components/StepUpload.vue'
import StepConfig from './components/StepConfig.vue'
import StepResults from './components/StepResults.vue'
import StepOutput from './components/StepOutput.vue'
import DocsTab from './components/DocsTab.vue'

const view = ref('app')
const step = ref(0)
const startMode = ref('file')
const dashboards = ref([])
const parseResult = ref(null)
const config = ref({
  stsUrl: '',
  stsToken: '',
  name: '',
  metricPrefix: '',
  rewriteMetrics: true,
  includeMissing: false,
})
const checkResult = ref(null)
const convertResult = ref(null)
const logs = ref([])

function log(msg, type = 'info') {
  const ts = new Date().toLocaleTimeString()
  logs.value.push({ ts, msg, type })
}

const steps = [
  { num: 1, label: 'Upload' },
  { num: 2, label: 'Configure' },
  { num: 3, label: 'Analyse' },
  { num: 4, label: 'Export' },
]

function start(mode = 'file') {
  startMode.value = mode
  step.value = 1
}

function onUploaded(data) {
  dashboards.value = data.dashboards
  parseResult.value = data.parseResult
  if (data.parseResult?.title) {
    config.value.name = data.parseResult.title
  }
  log(`Parsed "${data.parseResult?.title || data.dashboards[0]?.filename}" — ${data.parseResult?.panels?.length || 0} panels extracted`)
  step.value = 2
}

function onConfigured(cfg) {
  Object.assign(config.value, cfg)
  if (cfg.stsUrl) {
    log(`Connecting to SUSE Observability at ${cfg.stsUrl}`)
  } else {
    log('Proceeding without live metric check — converting all panels')
  }
  step.value = 3
}

function onChecked(result) {
  checkResult.value = result
  log(`Metric check complete: ${result.matched}/${result.panels?.length} panels matched (${result.totalMetrics.toLocaleString()} metrics in STS)`, result.matched > 0 ? 'success' : 'warn')
  if (result.detectedPrefix) {
    log(`Auto-detected metric prefix: "${result.detectedPrefix}"`)
  }
}

function onConvert() {
  log('Generating SUSE Observability dashboard YAML...')
  step.value = 4
}

function onConverted(result) {
  convertResult.value = result
  log(`YAML generated — ${result.panelCount} panels, ${(result.yaml?.length / 1024).toFixed(1)} KB`, 'success')
}

function reset() {
  step.value = 0
  dashboards.value = []
  parseResult.value = null
  checkResult.value = null
  convertResult.value = null
  logs.value = []
  config.value = { stsUrl: '', stsToken: '', name: '', metricPrefix: '', rewriteMetrics: true, includeMissing: false }
}
</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 text-white flex flex-col">
    <!-- Header -->
    <header class="border-b border-slate-800/60 backdrop-blur-sm bg-slate-900/50 sticky top-0 z-50">
      <div class="max-w-7xl mx-auto px-6 py-3.5 flex items-center justify-between">
        <div class="flex items-center cursor-pointer" @click="view = 'app'; step = 0">
          <h1 class="text-lg font-bold tracking-tight leading-tight">Odyssey</h1>
        </div>
        <div class="flex items-center gap-2">
          <button
            @click="view = view === 'docs' ? 'app' : 'docs'"
            :class="[
              'text-xs px-3 py-1.5 rounded-lg transition-all ring-1',
              view === 'docs'
                ? 'bg-emerald-500/15 text-emerald-400 ring-emerald-500/30'
                : 'text-slate-500 hover:text-white hover:bg-slate-800 ring-slate-700/50',
            ]"
          >
            <span class="flex items-center gap-1.5">
              <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 6.253v13m0-13C10.832 5.477 9.246 5 7.5 5S4.168 5.477 3 6.253v13C4.168 18.477 5.754 18 7.5 18s3.332.477 4.5 1.253m0-13C13.168 5.477 14.754 5 16.5 5c1.747 0 3.332.477 4.5 1.253v13C19.832 18.477 18.247 18 16.5 18c-1.746 0-3.332.477-4.5 1.253"/></svg>
              Docs
            </span>
          </button>
          <a href="https://github.com/aeltai/odyssey" target="_blank" class="text-slate-500 hover:text-slate-300 transition-colors p-1.5">
            <svg class="w-5 h-5" fill="currentColor" viewBox="0 0 24 24"><path d="M12 0C5.37 0 0 5.37 0 12c0 5.31 3.435 9.795 8.205 11.385.6.105.825-.255.825-.57 0-.285-.015-1.23-.015-2.235-3.015.555-3.795-.735-4.035-1.41-.135-.345-.72-1.41-1.23-1.695-.42-.225-1.02-.78-.015-.795.945-.015 1.62.87 1.845 1.23 1.08 1.815 2.805 1.305 3.495.99.105-.78.42-1.305.765-1.605-2.67-.3-5.46-1.335-5.46-5.925 0-1.305.465-2.385 1.23-3.225-.12-.3-.54-1.53.12-3.18 0 0 1.005-.315 3.3 1.23.96-.27 1.98-.405 3-.405s2.04.135 3 .405c2.295-1.56 3.3-1.23 3.3-1.23.66 1.65.24 2.88.12 3.18.765.84 1.23 1.905 1.23 3.225 0 4.605-2.805 5.625-5.475 5.925.435.375.81 1.095.81 2.22 0 1.605-.015 2.895-.015 3.3 0 .315.225.69.825.57A12.02 12.02 0 0024 12c0-6.63-5.37-12-12-12z"/></svg>
          </a>
          <button v-if="step > 0 && view === 'app'" @click="reset" class="text-xs text-slate-500 hover:text-white transition-colors px-3 py-1.5 rounded-lg hover:bg-slate-800 ring-1 ring-slate-700/50">
            Start over
          </button>
        </div>
      </div>
    </header>

    <!-- Documentation -->
    <template v-if="view === 'docs'">
      <DocsTab />
    </template>

    <!-- Landing -->
    <template v-else-if="step === 0">
      <div class="flex-1 flex flex-col items-center justify-center px-6 py-16">
        <div class="max-w-4xl mx-auto text-center space-y-10">
          <!-- Hero icons -->
          <div class="space-y-5">
            <div class="flex items-center justify-center gap-6 mb-6">
              <!-- Grafana -->
              <div class="w-16 h-16 rounded-2xl bg-white/5 flex items-center justify-center shadow-xl ring-1 ring-slate-700/50 p-3">
                <img src="https://cdn.jsdelivr.net/npm/simple-icons@v16/icons/grafana.svg" alt="Grafana" class="w-10 h-10 brightness-0 invert" />
              </div>
              <!-- Arrow -->
              <div class="flex items-center gap-2">
                <div class="w-12 h-px bg-gradient-to-r from-orange-500/50 to-emerald-500/50"></div>
                <div class="w-10 h-10 rounded-full bg-slate-800 ring-1 ring-slate-700/50 flex items-center justify-center">
                  <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" /></svg>
                </div>
                <div class="w-12 h-px bg-gradient-to-r from-emerald-500/50 to-teal-500/50"></div>
              </div>
              <!-- SUSE Observability -->
              <div class="w-16 h-16 rounded-2xl bg-white/5 flex items-center justify-center shadow-xl ring-1 ring-slate-700/50 p-3">
                <img src="https://cdn.jsdelivr.net/npm/simple-icons@v16/icons/suse.svg" alt="SUSE" class="w-10 h-10 brightness-0 invert" />
              </div>
            </div>

            <h2 class="text-4xl sm:text-5xl font-extrabold tracking-tight bg-gradient-to-r from-white via-slate-200 to-slate-400 bg-clip-text text-transparent">
              Migrate Grafana Dashboards to SUSE Observability
            </h2>
            <p class="text-lg text-slate-400 max-w-2xl mx-auto leading-relaxed">
              Connect directly to your Grafana instance or upload JSON exports. Validate metrics against a live SUSE Observability environment and get production-ready YAML.
            </p>
          </div>

          <!-- Two CTA cards -->
          <div class="grid sm:grid-cols-2 gap-4 max-w-2xl mx-auto">
            <!-- Connect to Grafana -->
            <button
              @click="start('grafana')"
              class="group bg-slate-800/40 hover:bg-orange-500/10 rounded-2xl ring-1 ring-slate-700/40 hover:ring-orange-500/30 p-6 text-left transition-all duration-200 space-y-3"
            >
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-xl bg-orange-500/15 flex items-center justify-center">
                  <img src="https://cdn.jsdelivr.net/npm/simple-icons@v16/icons/grafana.svg" alt="Grafana" class="w-5 h-5 brightness-0 invert opacity-90" />
                </div>
                <h3 class="font-semibold text-slate-200 group-hover:text-orange-300 transition-colors">Connect to Grafana</h3>
              </div>
              <p class="text-sm text-slate-500 leading-relaxed">
                Connect to your Grafana instance, browse dashboards, and pull them directly. Requires an API key or service account token.
              </p>
              <div class="flex items-center gap-1.5 text-xs text-orange-400/70 font-medium">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.828 10.172a4 4 0 00-5.656 0l-4 4a4 4 0 105.656 5.656l1.102-1.101m-.758-4.899a4 4 0 005.656 0l4-4a4 4 0 00-5.656-5.656l-1.1 1.1" /></svg>
                Pull dashboards live
              </div>
            </button>

            <!-- Upload JSON -->
            <button
              @click="start('file')"
              class="group bg-slate-800/40 hover:bg-emerald-500/10 rounded-2xl ring-1 ring-slate-700/40 hover:ring-emerald-500/30 p-6 text-left transition-all duration-200 space-y-3"
            >
              <div class="flex items-center gap-3">
                <div class="w-10 h-10 rounded-xl bg-emerald-500/15 flex items-center justify-center">
                  <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" /></svg>
                </div>
                <h3 class="font-semibold text-slate-200 group-hover:text-emerald-300 transition-colors">Upload JSON File</h3>
              </div>
              <p class="text-sm text-slate-500 leading-relaxed">
                Upload a Grafana dashboard JSON export. Go to Dashboard, Share, Export, Save to file in Grafana.
              </p>
              <div class="flex items-center gap-1.5 text-xs text-emerald-400/70 font-medium">
                <svg class="w-3.5 h-3.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
                Drag &amp; drop or browse
              </div>
            </button>
          </div>

          <!-- Feature pills -->
          <div class="flex flex-wrap items-center justify-center gap-3 text-xs">
            <span class="px-3 py-1.5 rounded-full bg-slate-800/50 ring-1 ring-slate-700/40 text-slate-400">PromQL Sanitization</span>
            <span class="px-3 py-1.5 rounded-full bg-slate-800/50 ring-1 ring-slate-700/40 text-slate-400">Metric Prefix Detection</span>
            <span class="px-3 py-1.5 rounded-full bg-slate-800/50 ring-1 ring-slate-700/40 text-slate-400">_total Suffix Handling</span>
            <span class="px-3 py-1.5 rounded-full bg-slate-800/50 ring-1 ring-slate-700/40 text-slate-400">Variable Replacement</span>
            <span class="px-3 py-1.5 rounded-full bg-slate-800/50 ring-1 ring-slate-700/40 text-slate-400">Direct Apply to STS</span>
          </div>

          <!-- Tested badge -->
          <p class="text-xs text-slate-600">
            Tested with 47 popular Grafana dashboards &middot; 1,551 panels &middot; Zero crashes
          </p>
        </div>
      </div>
    </template>

    <!-- Wizard -->
    <template v-else-if="view === 'app' && step > 0">
      <nav class="max-w-7xl mx-auto px-6 pt-6 pb-1">
        <div class="flex items-center gap-2">
          <template v-for="(s, i) in steps" :key="s.num">
            <button
              @click="s.num < step ? step = s.num : null"
              :class="[
                'flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium transition-all duration-300',
                step === s.num ? 'bg-emerald-500/20 text-emerald-400 ring-1 ring-emerald-500/30' : '',
                step > s.num ? 'text-slate-300 hover:text-white cursor-pointer' : '',
                step < s.num ? 'text-slate-600 cursor-default' : '',
              ]"
            >
              <span
                :class="[
                  'w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold transition-all',
                  step >= s.num ? 'bg-emerald-500 text-slate-900' : 'bg-slate-800 text-slate-600',
                ]"
              >
                <template v-if="step > s.num">&#10003;</template>
                <template v-else>{{ s.num }}</template>
              </span>
              {{ s.label }}
            </button>
            <div v-if="i < steps.length - 1" :class="['flex-1 h-px transition-colors', step > s.num ? 'bg-emerald-500/30' : 'bg-slate-800']" />
          </template>
        </div>
      </nav>

      <main class="max-w-7xl mx-auto px-6 py-6 flex-1">
        <div class="grid grid-cols-1 lg:grid-cols-3 gap-6">
          <!-- Main content -->
          <div class="lg:col-span-2">
            <Transition name="fade" mode="out-in">
              <StepUpload v-if="step === 1" :initial-mode="startMode" @uploaded="onUploaded" />
              <StepConfig v-else-if="step === 2" :parse-result="parseResult" :config="config" @configured="onConfigured" @back="step = 1" />
              <StepResults v-else-if="step === 3" :dashboards="dashboards" :config="config" @checked="onChecked" @convert="onConvert" @back="step = 2" />
              <StepOutput v-else-if="step === 4" :dashboards="dashboards" :config="config" :check-result="checkResult" @converted="onConverted" @back="step = 3" />
            </Transition>
          </div>

          <!-- Activity log sidebar -->
          <div class="hidden lg:block">
            <div class="sticky top-24 space-y-4">
              <div class="bg-slate-800/30 rounded-2xl ring-1 ring-slate-700/40 p-5">
                <h3 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">Activity Log</h3>
                <div v-if="logs.length === 0" class="text-sm text-slate-600 italic">No activity yet...</div>
                <div v-else class="space-y-2.5 max-h-64 overflow-y-auto">
                  <div v-for="(l, i) in [...logs].reverse()" :key="i" class="flex gap-2.5">
                    <div class="mt-1.5 flex-shrink-0">
                      <div :class="[
                        'w-2 h-2 rounded-full',
                        l.type === 'success' ? 'bg-emerald-400' : l.type === 'warn' ? 'bg-amber-400' : l.type === 'error' ? 'bg-red-400' : 'bg-slate-500',
                      ]" />
                    </div>
                    <div>
                      <p class="text-xs text-slate-400 leading-relaxed">{{ l.msg }}</p>
                      <p class="text-[10px] text-slate-600 mt-0.5">{{ l.ts }}</p>
                    </div>
                  </div>
                </div>
              </div>

              <!-- What happens -->
              <div class="bg-slate-800/30 rounded-2xl ring-1 ring-slate-700/40 p-5">
                <h3 class="text-xs font-semibold text-slate-500 uppercase tracking-wider mb-3">How It Works</h3>
                <div class="space-y-3">
                  <div class="flex gap-2.5">
                    <div class="w-5 h-5 rounded bg-orange-500/15 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span class="text-[10px] font-bold text-orange-400">1</span>
                    </div>
                    <p class="text-xs text-slate-500">Grafana JSON is parsed for all PromQL expressions</p>
                  </div>
                  <div class="flex gap-2.5">
                    <div class="w-5 h-5 rounded bg-emerald-500/15 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span class="text-[10px] font-bold text-emerald-400">2</span>
                    </div>
                    <p class="text-xs text-slate-500">Variables like <code class="text-slate-400">$__rate_interval</code> are replaced with safe defaults</p>
                  </div>
                  <div class="flex gap-2.5">
                    <div class="w-5 h-5 rounded bg-blue-500/15 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span class="text-[10px] font-bold text-blue-400">3</span>
                    </div>
                    <p class="text-xs text-slate-500">Metrics are checked against STS Prometheus API</p>
                  </div>
                  <div class="flex gap-2.5">
                    <div class="w-5 h-5 rounded bg-teal-500/15 flex items-center justify-center flex-shrink-0 mt-0.5">
                      <span class="text-[10px] font-bold text-teal-400">4</span>
                    </div>
                    <p class="text-xs text-slate-500">STS YAML is generated with proper layout, panels, and queries</p>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>
      </main>
    </template>

    <!-- Footer -->
    <footer class="border-t border-slate-800/60 mt-auto">
      <div class="max-w-7xl mx-auto px-6 py-5">
        <div class="flex flex-col sm:flex-row items-center justify-between gap-3">
          <div class="flex items-center gap-4 text-xs text-slate-600">
            <span>Odyssey v1.0</span>
            <span class="w-1 h-1 rounded-full bg-slate-700"></span>
            <span>Apache 2.0 License</span>
            <span class="w-1 h-1 rounded-full bg-slate-700"></span>
            <a href="https://github.com/aeltai/odyssey" target="_blank" class="hover:text-slate-400 transition-colors">GitHub</a>
          </div>
          <div class="flex items-center gap-3 text-xs text-slate-600">
            <img src="https://cdn.jsdelivr.net/npm/simple-icons@v16/icons/grafana.svg" alt="Grafana" class="w-4 h-4 brightness-0 invert opacity-70" />
            <svg class="w-3 h-3 text-slate-700" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13 7l5 5m0 0l-5 5m5-5H6" /></svg>
            <img src="https://cdn.jsdelivr.net/npm/simple-icons@v16/icons/suse.svg" alt="SUSE" class="w-4 h-4 brightness-0 invert opacity-70" />
          </div>
        </div>
      </div>
    </footer>
  </div>
</template>

<style>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.fade-enter-from { opacity: 0; transform: translateY(8px); }
.fade-leave-to { opacity: 0; transform: translateY(-8px); }
</style>
