<script setup>
import { ref, onMounted } from 'vue'

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

onMounted(async () => {
  await convert()
})

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
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
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

const lineCount = ref(0)
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-1">Generated YAML</h2>
      <p class="text-slate-400">Ready to apply to your SUSE Observability instance.</p>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-16">
      <div class="flex items-center gap-3 text-slate-400">
        <svg class="animate-spin w-5 h-5" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        <span>Generating YAML...</span>
      </div>
    </div>

    <template v-else-if="!error">
      <div class="grid grid-cols-3 gap-3">
        <div class="bg-emerald-500/10 rounded-xl ring-1 ring-emerald-500/20 p-4 text-center">
          <p class="text-2xl font-bold text-emerald-400">{{ panelCount }}</p>
          <p class="text-xs text-slate-400 mt-1">Panels Included</p>
        </div>
        <div class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/50 p-4 text-center">
          <p class="text-2xl font-bold text-white">{{ yaml.split('\n').length }}</p>
          <p class="text-xs text-slate-400 mt-1">YAML Lines</p>
        </div>
        <div v-if="detectedPrefix" class="bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20 p-4 text-center">
          <p class="text-lg font-bold text-blue-400 font-mono">{{ detectedPrefix }}</p>
          <p class="text-xs text-slate-400 mt-1">Prefix Used</p>
        </div>
        <div v-else class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/50 p-4 text-center">
          <p class="text-2xl font-bold text-white">{{ (yaml.length / 1024).toFixed(1) }}KB</p>
          <p class="text-xs text-slate-400 mt-1">File Size</p>
        </div>
      </div>

      <div class="flex items-center gap-2">
        <button
          @click="copyToClipboard"
          :class="[
            'flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium transition-all ring-1',
            copied
              ? 'bg-emerald-500/15 text-emerald-400 ring-emerald-500/30'
              : 'bg-slate-800 text-slate-300 hover:text-white ring-slate-700/50 hover:ring-slate-600',
          ]"
        >
          <svg v-if="!copied" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M8 16H6a2 2 0 01-2-2V6a2 2 0 012-2h8a2 2 0 012 2v2m-6 12h8a2 2 0 002-2v-8a2 2 0 00-2-2h-8a2 2 0 00-2 2v8a2 2 0 002 2z" />
          </svg>
          <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M5 13l4 4L19 7" />
          </svg>
          {{ copied ? 'Copied!' : 'Copy' }}
        </button>
        <button
          @click="download"
          class="flex items-center gap-2 px-4 py-2 rounded-xl text-sm font-medium bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25 transition-all"
        >
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
          </svg>
          Download .sts.yaml
        </button>
      </div>

      <div class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/50 p-4 text-center space-y-2">
        <p class="text-sm text-slate-400">Apply with the STS CLI:</p>
        <code class="block bg-slate-900 rounded-lg px-4 py-2.5 text-sm text-emerald-400 font-mono">
          sts dashboard apply --file {{ (config.name || 'dashboard').toLowerCase().replace(/\s+/g, '-') }}.sts.yaml
        </code>
      </div>

      <div class="relative">
        <div class="absolute top-3 right-3 text-[10px] text-slate-600 font-mono">YAML</div>
        <pre class="bg-slate-950 rounded-2xl ring-1 ring-slate-800 p-5 text-xs font-mono text-slate-300 overflow-auto max-h-[50vh] leading-relaxed"><code>{{ yaml }}</code></pre>
      </div>
    </template>

    <div v-if="error" class="bg-red-500/10 border border-red-500/30 rounded-xl px-4 py-3 text-red-400 text-sm">
      {{ error }}
    </div>

    <div class="flex gap-3">
      <button
        @click="$emit('back')"
        class="px-6 py-3 rounded-xl text-sm font-medium text-slate-400 hover:text-white bg-slate-800 hover:bg-slate-700 transition-all"
      >
        Back
      </button>
    </div>
  </div>
</template>
