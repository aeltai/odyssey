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
const expandedPanel = ref(null)

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

const warningCount = computed(() => panels.value.filter(p => p.warning).length)
const matchedCount = computed(() => panels.value.filter(p => p.hasData).length)
const missingCount = computed(() => hasSTSConnection.value ? panels.value.filter(p => !p.hasData && !p.warning).length : 0)

const donutSegments = computed(() => {
  const total = panels.value.length
  if (total === 0) return []
  const m = matchedCount.value
  const w = warningCount.value
  const miss = total - m - w
  const segments = []
  let offset = 0
  if (m > 0) { segments.push({ pct: (m / total) * 100, color: '#34d399', offset }); offset += (m / total) * 100 }
  if (miss > 0) { segments.push({ pct: (miss / total) * 100, color: '#64748b', offset }); offset += (miss / total) * 100 }
  if (w > 0) { segments.push({ pct: (w / total) * 100, color: '#fbbf24', offset }); offset += (w / total) * 100 }
  return segments
})

onMounted(async () => {
  if (hasSTSConnection.value) await checkMetrics()
  else await parseOnly()
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
    panels.value = data.panels.map(p => ({ ...p, hasData: true }))
    matched.value = data.panels.length
    missing.value = 0
  } catch (e) { error.value = e.message }
  finally { loading.value = false }
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
  } catch (e) { error.value = e.message }
  finally { loading.value = false }
}

function togglePanel(i) {
  expandedPanel.value = expandedPanel.value === i ? null : i
}

const filterCounts = computed(() => ({
  all: panels.value.length,
  matched: matchedCount.value,
  missing: missingCount.value,
  warnings: warningCount.value,
}))
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-2">Panel Analysis</h2>
      <p class="text-slate-400 leading-relaxed">
        <template v-if="hasSTSConnection">
          Metrics checked against your SUSE Observability Prometheus endpoint. Panels with missing metrics can still be included in the output.
        </template>
        <template v-else>
          All panels parsed and PromQL sanitised. Connect to SUSE Observability in the previous step for live metric matching.
        </template>
      </p>
    </div>

    <!-- Loading -->
    <div v-if="loading" class="flex flex-col items-center justify-center py-20 gap-4">
      <div class="relative">
        <div class="w-16 h-16 rounded-full border-4 border-slate-800"></div>
        <div class="w-16 h-16 rounded-full border-4 border-t-emerald-400 border-r-transparent border-b-transparent border-l-transparent absolute top-0 left-0 animate-spin"></div>
      </div>
      <div class="text-center">
        <p class="text-slate-300 font-medium">{{ hasSTSConnection ? 'Checking metrics...' : 'Parsing panels...' }}</p>
        <p class="text-xs text-slate-600 mt-1">{{ hasSTSConnection ? 'Querying STS Prometheus API for each metric' : 'Extracting PromQL from all panel types' }}</p>
      </div>
    </div>

    <template v-else-if="!error">
      <!-- Stats row -->
      <div class="flex gap-4 items-start">
        <!-- Donut chart (only with STS) -->
        <div v-if="hasSTSConnection && panels.length > 0" class="flex-shrink-0">
          <div class="relative w-28 h-28">
            <svg class="w-full h-full -rotate-90" viewBox="0 0 36 36">
              <circle cx="18" cy="18" r="15.9" fill="none" stroke="#1e293b" stroke-width="3" />
              <circle
                v-for="(seg, i) in donutSegments"
                :key="i"
                cx="18" cy="18" r="15.9"
                fill="none"
                :stroke="seg.color"
                stroke-width="3"
                stroke-linecap="round"
                :stroke-dasharray="`${seg.pct} ${100 - seg.pct}`"
                :stroke-dashoffset="`${-seg.offset}`"
                class="transition-all duration-700"
              />
            </svg>
            <div class="absolute inset-0 flex flex-col items-center justify-center">
              <p class="text-xl font-bold text-white leading-none">{{ matchPercent }}%</p>
              <p class="text-[9px] text-slate-500 mt-0.5">matched</p>
            </div>
          </div>
        </div>

        <!-- Stat cards -->
        <div class="flex-1 grid grid-cols-2 sm:grid-cols-3 gap-2.5">
          <div class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/40 p-3.5 text-center">
            <p class="text-xl font-bold text-white">{{ panels.length }}</p>
            <p class="text-[10px] text-slate-500 mt-0.5">Total Panels</p>
          </div>
          <div v-if="hasSTSConnection" class="bg-emerald-500/8 rounded-xl ring-1 ring-emerald-500/20 p-3.5 text-center">
            <p class="text-xl font-bold text-emerald-400">{{ matched }}</p>
            <p class="text-[10px] text-slate-500 mt-0.5">Matched</p>
          </div>
          <div v-if="hasSTSConnection" class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/40 p-3.5 text-center">
            <p class="text-xl font-bold text-slate-400">{{ missing }}</p>
            <p class="text-[10px] text-slate-500 mt-0.5">Missing</p>
          </div>
          <div v-if="detectedPrefix" class="bg-blue-500/8 rounded-xl ring-1 ring-blue-500/20 p-3.5 text-center">
            <p class="text-base font-bold text-blue-400 font-mono truncate">{{ detectedPrefix }}</p>
            <p class="text-[10px] text-slate-500 mt-0.5">Prefix</p>
          </div>
          <div v-if="hasSTSConnection" class="bg-slate-800/40 rounded-xl ring-1 ring-slate-700/40 p-3.5 text-center">
            <p class="text-xl font-bold text-white">{{ totalMetrics.toLocaleString() }}</p>
            <p class="text-[10px] text-slate-500 mt-0.5">STS Metrics</p>
          </div>
          <div v-if="warningCount" class="bg-amber-500/8 rounded-xl ring-1 ring-amber-500/20 p-3.5 text-center">
            <p class="text-xl font-bold text-amber-400">{{ warningCount }}</p>
            <p class="text-[10px] text-slate-500 mt-0.5">Warnings</p>
          </div>
        </div>
      </div>

      <!-- Progress bar -->
      <div v-if="hasSTSConnection && panels.length > 0" class="space-y-1.5">
        <div class="w-full bg-slate-800 rounded-full h-2 overflow-hidden">
          <div class="h-full rounded-full transition-all duration-700 bg-gradient-to-r from-emerald-500 to-teal-400" :style="{ width: matchPercent + '%' }" />
        </div>
        <p class="text-[10px] text-slate-600 text-right">{{ matched }} of {{ panels.length }} panels have matching metrics</p>
      </div>

      <!-- Filters -->
      <div class="flex items-center gap-3 flex-wrap">
        <div class="flex bg-slate-800/40 rounded-lg p-0.5 ring-1 ring-slate-700/40">
          <button
            v-for="f in [
              { key: 'all', label: 'All', icon: null },
              { key: 'matched', label: 'Matched', icon: null },
              { key: 'missing', label: 'Missing', icon: null },
              { key: 'warnings', label: 'Warnings', icon: null },
            ]"
            :key="f.key"
            @click="filter = f.key"
            :class="[
              'px-3 py-1.5 text-xs font-medium rounded-md transition-all flex items-center gap-1.5',
              filter === f.key ? 'bg-slate-700 text-white shadow-sm' : 'text-slate-500 hover:text-slate-300',
            ]"
          >
            {{ f.label }}
            <span v-if="filterCounts[f.key] > 0" :class="['text-[10px] px-1.5 py-0.5 rounded-full', filter === f.key ? 'bg-slate-600 text-slate-200' : 'bg-slate-800 text-slate-500']">{{ filterCounts[f.key] }}</span>
          </button>
        </div>
        <div class="flex-1">
          <input
            v-model="search"
            type="text"
            placeholder="Search panels or metrics..."
            class="w-full bg-slate-900/60 border border-slate-700/60 rounded-lg px-3 py-1.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/30 transition"
          />
        </div>
      </div>

      <!-- Panel list -->
      <div class="space-y-2 max-h-[55vh] overflow-y-auto scrollbar-thin pr-1">
        <div
          v-for="(p, i) in filteredPanels"
          :key="i"
          :class="[
            'rounded-xl ring-1 transition-all duration-200 cursor-pointer',
            p.warning ? 'bg-amber-500/5 ring-amber-500/15 hover:ring-amber-500/30' :
            p.hasData ? 'bg-emerald-500/3 ring-emerald-500/15 hover:ring-emerald-500/30' :
            'bg-slate-800/20 ring-slate-700/30 hover:ring-slate-600/50',
          ]"
          @click="togglePanel(i)"
        >
          <div class="px-4 py-3">
            <div class="flex items-start justify-between gap-3">
              <div class="min-w-0 flex-1">
                <div class="flex items-center gap-2 mb-1">
                  <span
                    v-if="hasSTSConnection"
                    :class="[
                      'inline-block w-2 h-2 rounded-full flex-shrink-0',
                      p.warning ? 'bg-amber-400' : p.hasData ? 'bg-emerald-400' : 'bg-slate-600',
                    ]"
                  />
                  <p class="text-sm font-medium text-slate-200 truncate">{{ p.title }}</p>
                  <span v-if="p.warning" class="text-[10px] bg-amber-500/15 text-amber-400 px-2 py-0.5 rounded-full font-medium flex-shrink-0">{{ p.warning }}</span>
                </div>
                <p class="text-xs text-slate-600 font-mono truncate">{{ p.sanitized }}</p>
              </div>
              <svg :class="['w-4 h-4 text-slate-600 transition-transform flex-shrink-0 mt-1', expandedPanel === i ? 'rotate-180' : '']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M19 9l-7 7-7-7" /></svg>
            </div>
          </div>

          <!-- Expanded details -->
          <Transition name="expand">
            <div v-if="expandedPanel === i" class="px-4 pb-3 space-y-2 border-t border-slate-700/30 pt-2">
              <div v-if="p.original && p.original !== p.sanitized" class="space-y-1">
                <p class="text-[10px] text-slate-600 uppercase tracking-wider font-semibold">Original PromQL</p>
                <p class="text-xs text-slate-500 font-mono bg-slate-900/50 rounded-lg px-3 py-2 break-all">{{ p.original }}</p>
              </div>
              <div class="space-y-1">
                <p class="text-[10px] text-slate-600 uppercase tracking-wider font-semibold">Sanitised PromQL</p>
                <p class="text-xs text-slate-400 font-mono bg-slate-900/50 rounded-lg px-3 py-2 break-all">{{ p.sanitized }}</p>
              </div>
              <div v-if="p.metrics?.length" class="space-y-1">
                <p class="text-[10px] text-slate-600 uppercase tracking-wider font-semibold">Extracted Metrics</p>
                <div class="flex flex-wrap gap-1.5">
                  <span
                    v-for="m in p.metrics"
                    :key="m"
                    :class="[
                      'inline-block px-2 py-0.5 text-[10px] font-mono rounded-md ring-1',
                      p.hasData ? 'bg-emerald-500/10 text-emerald-400 ring-emerald-500/20' : 'bg-slate-800 text-slate-500 ring-slate-700/50',
                    ]"
                  >{{ m }}</span>
                </div>
              </div>
            </div>
          </Transition>
        </div>
        <p v-if="filteredPanels.length === 0" class="text-center text-sm text-slate-600 py-8">No panels match your filter.</p>
      </div>
    </template>

    <div v-if="error" class="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-red-400 text-sm flex items-start gap-2">
      <svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      {{ error }}
    </div>

    <div class="flex gap-3">
      <button @click="$emit('back')" class="px-5 py-3 rounded-xl text-sm font-medium text-slate-400 hover:text-white bg-slate-800/60 hover:bg-slate-700/60 ring-1 ring-slate-700/40 transition-all">
        Back
      </button>
      <button
        @click="$emit('convert')"
        :disabled="loading || panels.length === 0"
        :class="[
          'flex-1 py-3 rounded-xl font-semibold text-sm transition-all flex items-center justify-center gap-2',
          !loading && panels.length > 0
            ? 'bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25'
            : 'bg-slate-800/60 text-slate-600 cursor-not-allowed',
        ]"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
        Generate STS YAML
      </button>
    </div>
  </div>
</template>

<style scoped>
.expand-enter-active, .expand-leave-active { transition: all 0.2s ease; }
.expand-enter-from, .expand-leave-to { opacity: 0; max-height: 0; }
.expand-enter-to, .expand-leave-from { opacity: 1; max-height: 500px; }
</style>
