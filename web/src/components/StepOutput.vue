<script setup>
import { ref, onMounted, computed } from 'vue'

const props = defineProps(['dashboards', 'config', 'checkResult'])
const emit = defineEmits(['converted', 'back'])

const yaml = ref('')
const loading = ref(false)
const error = ref('')
const panelCount = ref(0)
const matched = ref(0)
const missing = ref(0)
const detectedPrefix = ref('')
const copied = ref(false)
const showApplyGuide = ref(false)

const applying = ref(false)
const applyResult = ref(null)
const applyError = ref('')

const hasSTSConnection = computed(() => !!props.config.stsUrl && !!props.config.stsToken)

onMounted(async () => { await convert() })

async function convert() {
  loading.value = true
  error.value = ''
  try {
    const resp = await fetch('/api/convert', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        dashboards: props.dashboards,
        stsUrl: props.config.stsUrl,
        stsToken: props.config.stsToken,
        name: props.config.name,
        metricPrefix: props.config.metricPrefix,
        rewriteMetrics: props.config.rewriteMetrics,
        includeMissing: props.config.includeMissing,
      }),
    })
    const data = await resp.json()
    if (!resp.ok) throw new Error(data.error)
    yaml.value = data.yaml
    panelCount.value = data.panelCount
    matched.value = data.matched
    missing.value = data.missing
    detectedPrefix.value = data.detectedPrefix
    emit('converted', data)
  } catch (e) { error.value = e.message }
  finally { loading.value = false }
}

async function applyToSTS() {
  applying.value = true
  applyError.value = ''
  applyResult.value = null
  try {
    const resp = await fetch('/api/apply', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        yaml: yaml.value,
        stsUrl: props.config.stsUrl,
        stsToken: props.config.stsToken,
      }),
    })
    const data = await resp.json()
    if (!resp.ok) throw new Error(data.error)
    applyResult.value = data
  } catch (e) { applyError.value = e.message }
  finally { applying.value = false }
}

function copyToClipboard() {
  navigator.clipboard.writeText(yaml.value)
  copied.value = true
  setTimeout(() => (copied.value = false), 2000)
}

function download() {
  const name = (props.config.name || 'dashboard').toLowerCase().replace(/\s+/g, '-')
  const blob = new Blob([yaml.value], { type: 'application/x-yaml' })
  const url = URL.createObjectURL(blob)
  const a = document.createElement('a')
  a.href = url
  a.download = `${name}.sts.yaml`
  a.click()
  URL.revokeObjectURL(url)
}

const yamlLines = computed(() => yaml.value.split('\n').length)
const yamlSize = computed(() => (yaml.value.length / 1024).toFixed(1))
const fileName = computed(() => `${(props.config.name || 'dashboard').toLowerCase().replace(/\s+/g, '-')}.sts.yaml`)
const numberedYaml = computed(() => {
  if (!yaml.value) return []
  return yaml.value.split('\n').map((line, i) => ({ num: i + 1, content: line }))
})
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-2">Export Dashboard</h2>
      <p class="text-slate-400 leading-relaxed">
        Your SUSE Observability dashboard is ready. Apply it directly, download the YAML, or copy the contents.
      </p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20 gap-4">
      <div class="relative">
        <div class="w-16 h-16 rounded-full border-4 border-slate-800"></div>
        <div class="w-16 h-16 rounded-full border-4 border-t-emerald-400 border-r-transparent border-b-transparent border-l-transparent absolute top-0 left-0 animate-spin"></div>
      </div>
      <p class="text-slate-300 font-medium">Converting dashboard to STS format...</p>
    </div>

    <template v-else-if="!error">
      <!-- Stats -->
      <div class="grid grid-cols-4 gap-2.5">
        <div class="bg-emerald-500/8 rounded-xl ring-1 ring-emerald-500/20 p-3.5 text-center">
          <p class="text-xl font-bold text-emerald-400">{{ panelCount }}</p>
          <p class="text-[10px] text-slate-500 mt-0.5">Panels</p>
        </div>
        <div class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/40 p-3.5 text-center">
          <p class="text-xl font-bold text-white">{{ yamlLines }}</p>
          <p class="text-[10px] text-slate-500 mt-0.5">Lines</p>
        </div>
        <div class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/40 p-3.5 text-center">
          <p class="text-xl font-bold text-white">{{ yamlSize }} KB</p>
          <p class="text-[10px] text-slate-500 mt-0.5">Size</p>
        </div>
        <div v-if="detectedPrefix" class="bg-blue-500/8 rounded-xl ring-1 ring-blue-500/20 p-3.5 text-center">
          <p class="text-base font-bold text-blue-400 font-mono truncate">{{ detectedPrefix }}</p>
          <p class="text-[10px] text-slate-500 mt-0.5">Prefix</p>
        </div>
        <div v-else class="bg-emerald-500/8 rounded-xl ring-1 ring-emerald-500/20 p-3.5 text-center">
          <svg class="w-5 h-5 mx-auto text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
          <p class="text-[10px] text-slate-500 mt-0.5">Ready</p>
        </div>
      </div>

      <!-- Apply to STS (prominent if connected) -->
      <div v-if="hasSTSConnection" class="bg-emerald-500/5 rounded-2xl ring-1 ring-emerald-500/20 p-5 space-y-3">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-emerald-500/15 flex items-center justify-center">
            <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3l14 9-14 9V3z" /></svg>
          </div>
          <div>
            <h3 class="text-sm font-semibold text-emerald-300">Apply Directly to SUSE Observability</h3>
            <p class="text-xs text-slate-500">Push this dashboard to {{ config.stsUrl }} using the Dashboards API</p>
          </div>
        </div>

        <div v-if="applyResult" class="bg-emerald-500/10 rounded-xl px-4 py-3 flex items-start gap-2">
          <svg class="w-5 h-5 text-emerald-400 flex-shrink-0 mt-0.5" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          <div>
            <p class="text-sm text-emerald-300 font-medium">{{ applyResult.message }}</p>
            <p v-if="applyResult.dashboardId" class="text-xs text-slate-400 mt-0.5">Dashboard ID: {{ applyResult.dashboardId }} &middot; <a :href="config.stsUrl" target="_blank" class="text-emerald-400 hover:underline">Open in STS</a></p>
          </div>
        </div>

        <div v-if="applyError" class="bg-red-500/10 rounded-xl px-4 py-3 text-red-400 text-sm flex items-start gap-2">
          <svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
          {{ applyError }}
        </div>

        <button
          @click="applyToSTS"
          :disabled="applying || !!applyResult"
          :class="[
            'w-full py-3 rounded-xl font-semibold text-sm transition-all flex items-center justify-center gap-2',
            applyResult ? 'bg-emerald-500/20 text-emerald-400 cursor-default' :
            applying ? 'bg-slate-800/60 text-slate-500 cursor-wait' :
            'bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25'
          ]"
        >
          <template v-if="applying">
            <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
            Applying to STS...
          </template>
          <template v-else-if="applyResult">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
            Applied Successfully
          </template>
          <template v-else>
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 3l14 9-14 9V3z" /></svg>
            Apply Dashboard to STS
          </template>
        </button>
      </div>

      <!-- Download / Copy actions -->
      <div class="flex items-center gap-2">
        <button @click="download" class="flex items-center gap-2 px-5 py-2.5 rounded-xl text-sm font-semibold bg-slate-700 hover:bg-slate-600 text-white ring-1 ring-slate-600/50 transition-all">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
          Download {{ fileName }}
        </button>
        <button @click="copyToClipboard" :class="['flex items-center gap-2 px-4 py-2.5 rounded-xl text-sm font-medium transition-all ring-1', copied ? 'bg-emerald-500/15 text-emerald-400 ring-emerald-500/30' : 'bg-slate-800/60 text-slate-300 hover:text-white ring-slate-700/40 hover:ring-slate-600']">
          <svg v-if="!copied" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" /></svg>
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" /></svg>
          {{ copied ? 'Copied!' : 'Copy YAML' }}
        </button>
      </div>

      <!-- Manual apply guide (collapsed by default, only shown when no STS connection) -->
      <div v-if="!hasSTSConnection" class="bg-slate-800/20 rounded-2xl ring-1 ring-slate-700/40 overflow-hidden">
        <button @click="showApplyGuide = !showApplyGuide" class="w-full flex items-center justify-between px-5 py-4 hover:bg-slate-800/30 transition-colors">
          <div class="flex items-center gap-3">
            <div class="w-8 h-8 rounded-lg bg-teal-500/15 flex items-center justify-center">
              <svg class="w-4 h-4 text-teal-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 9l3 3-3 3m5 0h3M5 20h14a2 2 0 002-2V6a2 2 0 00-2-2H5a2 2 0 00-2 2v12a2 2 0 002 2z" /></svg>
            </div>
            <span class="text-sm font-medium text-slate-200">How to apply this dashboard manually</span>
          </div>
          <svg :class="['w-4 h-4 text-slate-500 transition-transform', showApplyGuide ? 'rotate-180' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
        </button>
        <div v-if="showApplyGuide" class="px-5 pb-5 space-y-4 border-t border-slate-700/30">
          <div class="mt-4 space-y-3">
            <div class="flex gap-3">
              <div class="w-6 h-6 rounded-full bg-emerald-500/15 flex items-center justify-center flex-shrink-0 mt-0.5"><span class="text-[10px] font-bold text-emerald-400">1</span></div>
              <div>
                <p class="text-sm text-slate-300 font-medium">Install the STS CLI</p>
                <code class="text-xs text-slate-500 font-mono mt-1 block bg-slate-900/60 rounded-lg px-3 py-2">curl -sSL https://dl.stackstate.com/sts-cli/install.sh | bash</code>
              </div>
            </div>
            <div class="flex gap-3">
              <div class="w-6 h-6 rounded-full bg-emerald-500/15 flex items-center justify-center flex-shrink-0 mt-0.5"><span class="text-[10px] font-bold text-emerald-400">2</span></div>
              <div>
                <p class="text-sm text-slate-300 font-medium">Configure your instance</p>
                <code class="text-xs text-slate-500 font-mono mt-1 block bg-slate-900/60 rounded-lg px-3 py-2">sts context save --name prod --url https://your-sts-instance.com --api-token &lt;token&gt;</code>
              </div>
            </div>
            <div class="flex gap-3">
              <div class="w-6 h-6 rounded-full bg-emerald-500/15 flex items-center justify-center flex-shrink-0 mt-0.5"><span class="text-[10px] font-bold text-emerald-400">3</span></div>
              <div>
                <p class="text-sm text-slate-300 font-medium">Apply the dashboard</p>
                <code class="text-xs text-emerald-400 font-mono mt-1 block bg-slate-900/60 rounded-lg px-3 py-2">sts dashboard apply --file {{ fileName }}</code>
              </div>
            </div>
          </div>
        </div>
      </div>

      <!-- YAML preview -->
      <div>
        <div class="flex items-center justify-between mb-2">
          <p class="text-xs text-slate-500 font-medium">YAML Preview</p>
          <p class="text-[10px] text-slate-600 font-mono">{{ fileName }}</p>
        </div>
        <div class="bg-slate-950/80 rounded-2xl ring-1 ring-slate-800/80 overflow-hidden">
          <div class="max-h-[45vh] overflow-auto">
            <table class="w-full text-xs font-mono">
              <tbody>
                <tr v-for="line in numberedYaml" :key="line.num" class="hover:bg-slate-800/30">
                  <td class="py-0 px-3 text-right text-slate-700 select-none w-12 sticky left-0 bg-slate-950/80 border-r border-slate-800/50">{{ line.num }}</td>
                  <td class="py-0 px-4 text-slate-300 whitespace-pre">{{ line.content }}</td>
                </tr>
              </tbody>
            </table>
          </div>
        </div>
      </div>

      <!-- Source → Target -->
      <div class="grid grid-cols-2 gap-3">
        <div class="bg-slate-800/20 rounded-xl ring-1 ring-slate-700/40 p-4">
          <div class="flex items-center gap-2 mb-2">
            <div class="w-6 h-6 rounded bg-orange-500/15 flex items-center justify-center">
              <svg class="w-3.5 h-3.5 text-orange-400" viewBox="0 0 24 24" fill="currentColor"><path d="M22.687 12.566c-.045-.498-.18-.907-.315-1.316A8.76 8.76 0 0019.381 7.5C17.5 5.5 15 4.5 12 4.5S6.5 5.5 4.619 7.5A8.76 8.76 0 001.718 11.25c-.135.409-.27.818-.315 1.316C1.358 12.793 1.358 13.019 1.358 13.245s0 .452.045.679c.045.498.18.907.315 1.316A8.76 8.76 0 004.619 18.5C6.5 20.5 9 21.5 12 21.5s5.5-1 7.381-3A8.76 8.76 0 0022.372 15.24c.135-.41.27-.818.315-1.316.045-.227.045-.453.045-.68s0-.452-.045-.679zM12 19.43c-3.555 0-6.43-2.875-6.43-6.43S8.445 6.57 12 6.57s6.43 2.875 6.43 6.43-2.875 6.43-6.43 6.43z"/></svg>
            </div>
            <p class="text-xs font-semibold text-slate-400">Source</p>
          </div>
          <p class="text-sm text-slate-300">Grafana Dashboard</p>
          <p class="text-xs text-slate-600 mt-0.5">PromQL queries extracted and sanitised</p>
        </div>
        <div class="bg-slate-800/20 rounded-xl ring-1 ring-slate-700/40 p-4">
          <div class="flex items-center gap-2 mb-2">
            <div class="w-6 h-6 rounded bg-teal-500/15 flex items-center justify-center">
              <svg class="w-3.5 h-3.5 text-teal-400" viewBox="0 0 24 24" fill="currentColor"><path d="M12 2C6.48 2 2 6.48 2 12s4.48 10 10 10 10-4.48 10-10S17.52 2 12 2zm0 18c-4.41 0-8-3.59-8-8s3.59-8 8-8 8 3.59 8 8-3.59 8-8 8z"/></svg>
            </div>
            <p class="text-xs font-semibold text-slate-400">Target</p>
          </div>
          <p class="text-sm text-slate-300">SUSE Observability</p>
          <p class="text-xs text-slate-600 mt-0.5">{{ hasSTSConnection ? 'Connected — ready to apply' : 'Download & apply with sts CLI' }}</p>
        </div>
      </div>
    </template>

    <div v-if="error" class="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-red-400 text-sm flex items-start gap-2">
      <svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      {{ error }}
    </div>

    <div class="flex gap-3">
      <button @click="$emit('back')" class="px-5 py-3 rounded-xl text-sm font-medium text-slate-400 hover:text-white bg-slate-800/60 hover:bg-slate-700/60 ring-1 ring-slate-700/40 transition-all">
        Back to Analysis
      </button>
    </div>
  </div>
</template>
