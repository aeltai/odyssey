<script setup>
import { ref, onMounted, computed } from 'vue'

const props = defineProps(['dashboards', 'config'])
const emit = defineEmits(['checked', 'convert', 'back'])

const panels = ref([])
const loading = ref(false)
const error = ref('')
const totalMetrics = ref(0)
const matched = ref(0)
const missing = ref(0)
const detectedPrefix = ref('')
const hasSTSConnection = computed(() => !!props.config.stsUrl && !!props.config.stsToken)
const filter = ref('all')
const search = ref('')

const filteredPanels = computed(() => {
  let list = panels.value
  if (filter.value === 'matched') list = list.filter(p => p.hasData)
  else if (filter.value === 'missing') list = list.filter(p => !p.hasData && !p.warning)
  else if (filter.value === 'warnings') list = list.filter(p => p.warning)
  if (search.value) {
    const q = search.value.toLowerCase()
    list = list.filter(p => p.title.toLowerCase().includes(q) || p.sanitized.toLowerCase().includes(q))
  }
  return list
})

const matchPercent = computed(() => {
  if (panels.value.length === 0) return 0
  return Math.round((matched.value / panels.value.length) * 100)
})

onMounted(async () => {
  if (hasSTSConnection.value) {
    await checkMetrics()
  } else {
    await parseOnly()
  }
})

async function parseOnly() {
  loading.value = true
  try {
    const resp = await fetch('/api/parse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(props.dashboards[0]),
    })
    const data = await resp.json()
    if (!resp.ok) throw new Error(data.error)
    panels.value = data.panels
    matched.value = data.panels.length
    missing.value = 0
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

async function checkMetrics() {
  loading.value = true
  error.value = ''
  try {
    const resp = await fetch('/api/check', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({
        dashboards: props.dashboards,
        stsUrl: props.config.stsUrl,
        stsToken: props.config.stsToken,
      }),
    })
    const data = await resp.json()
    if (!resp.ok) throw new Error(data.error)
    panels.value = data.panels
    totalMetrics.value = data.totalMetrics
    matched.value = data.matched
    missing.value = data.missing
    detectedPrefix.value = data.detectedPrefix
    emit('checked', data)
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-1">Panel Analysis</h2>
      <p class="text-slate-400">
        <template v-if="hasSTSConnection">Metric availability checked against your SUSE Observability instance.</template>
        <template v-else>Panels parsed and sanitised. Connect STS for metric matching.</template>
      </p>
    </div>

    <div v-if="loading" class="flex items-center justify-center py-16">
      <div class="flex items-center gap-3 text-slate-400">
        <svg class="animate-spin w-5 h-5" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        <span>{{ hasSTSConnection ? 'Checking metrics...' : 'Parsing panels...' }}</span>
      </div>
    </div>

    <template v-else-if="!error">
      <div class="grid grid-cols-2 sm:grid-cols-4 gap-3">
        <div class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/50 p-4 text-center">
          <p class="text-2xl font-bold text-white">{{ panels.length }}</p>
          <p class="text-xs text-slate-400 mt-1">Total Panels</p>
        </div>
        <div v-if="hasSTSConnection" class="bg-emerald-500/10 rounded-xl ring-1 ring-emerald-500/20 p-4 text-center">
          <p class="text-2xl font-bold text-emerald-400">{{ matched }}</p>
          <p class="text-xs text-slate-400 mt-1">Matched</p>
        </div>
        <div v-if="hasSTSConnection" class="bg-amber-500/10 rounded-xl ring-1 ring-amber-500/20 p-4 text-center">
          <p class="text-2xl font-bold text-amber-400">{{ missing }}</p>
          <p class="text-xs text-slate-400 mt-1">Missing</p>
        </div>
        <div v-if="detectedPrefix" class="bg-blue-500/10 rounded-xl ring-1 ring-blue-500/20 p-4 text-center">
          <p class="text-lg font-bold text-blue-400 font-mono">{{ detectedPrefix }}</p>
          <p class="text-xs text-slate-400 mt-1">Prefix</p>
        </div>
        <div v-if="hasSTSConnection" class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/50 p-4 text-center">
          <p class="text-2xl font-bold text-white">{{ totalMetrics.toLocaleString() }}</p>
          <p class="text-xs text-slate-400 mt-1">STS Metrics</p>
        </div>
      </div>

      <div v-if="hasSTSConnection && panels.length > 0" class="w-full bg-slate-800 rounded-full h-2.5 overflow-hidden">
        <div
          class="h-full rounded-full transition-all duration-500 bg-gradient-to-r from-emerald-500 to-teal-400"
          :style="{ width: matchPercent + '%' }"
        />
      </div>

      <div class="flex items-center gap-3">
        <div class="flex bg-slate-800/60 rounded-lg p-0.5 ring-1 ring-slate-700/50">
          <button
            v-for="f in [
              { key: 'all', label: 'All' },
              { key: 'matched', label: 'Matched' },
              { key: 'missing', label: 'Missing' },
              { key: 'warnings', label: 'Warnings' },
            ]"
            :key="f.key"
            @click="filter = f.key"
            :class="[
              'px-3 py-1.5 text-xs font-medium rounded-md transition-all',
              filter === f.key ? 'bg-slate-700 text-white' : 'text-slate-400 hover:text-slate-200',
            ]"
          >
            {{ f.label }}
          </button>
        </div>
        <input
          v-model="search"
          type="text"
          placeholder="Search panels..."
          class="flex-1 bg-slate-900 border border-slate-700 rounded-lg px-3 py-1.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 transition"
        />
      </div>

      <div class="space-y-2 max-h-[50vh] overflow-y-auto pr-1">
        <div
          v-for="(p, i) in filteredPanels"
          :key="i"
          :class="[
            'rounded-xl ring-1 px-4 py-3 transition-all',
            p.warning ? 'bg-amber-500/5 ring-amber-500/20' :
            p.hasData ? 'bg-emerald-500/5 ring-emerald-500/20' :
            hasSTSConnection ? 'bg-slate-800/30 ring-slate-700/30' : 'bg-slate-800/30 ring-slate-700/30',
          ]"
        >
          <div class="flex items-start justify-between gap-3">
            <div class="min-w-0 flex-1">
              <div class="flex items-center gap-2 mb-1">
                <span
                  v-if="hasSTSConnection"
                  :class="[
                    'inline-block w-2 h-2 rounded-full flex-shrink-0',
                    p.warning ? 'bg-amber-400' : p.hasData ? 'bg-emerald-400' : 'bg-slate-500',
                  ]"
                />
                <p class="text-sm font-medium text-slate-200 truncate">{{ p.title }}</p>
              </div>
              <p class="text-xs text-slate-500 font-mono truncate">{{ p.sanitized }}</p>
              <div v-if="p.metrics.length" class="flex flex-wrap gap-1.5 mt-2">
                <span
                  v-for="m in p.metrics"
                  :key="m"
                  class="inline-block px-2 py-0.5 text-[10px] font-mono rounded-md bg-slate-800 text-slate-400 ring-1 ring-slate-700/50"
                >
                  {{ m }}
                </span>
              </div>
            </div>
            <span v-if="p.warning" class="text-xs text-amber-400 flex-shrink-0 mt-0.5">{{ p.warning }}</span>
          </div>
        </div>
        <p v-if="filteredPanels.length === 0" class="text-center text-sm text-slate-500 py-8">No panels match your filter.</p>
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
      <button
        @click="$emit('convert')"
        :disabled="loading || panels.length === 0"
        :class="[
          'flex-1 py-3 rounded-xl font-semibold text-sm transition-all',
          !loading && panels.length > 0
            ? 'bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25'
            : 'bg-slate-800 text-slate-500 cursor-not-allowed',
        ]"
      >
        Generate STS YAML
      </button>
    </div>
  </div>
</template>
