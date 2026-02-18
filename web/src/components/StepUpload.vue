<script setup>
import { ref, computed } from 'vue'

const props = defineProps({
  initialMode: { type: String, default: 'file' },
})
const emit = defineEmits(['uploaded'])

const mode = ref(props.initialMode)
const files = ref([])
const dragging = ref(false)
const error = ref('')
const loading = ref(false)

// Grafana connection
const grafanaUrl = ref('')
const grafanaToken = ref('')
const grafanaDashboards = ref([])
const grafanaLoading = ref(false)
const grafanaSearch = ref('')
const selectedDashboard = ref(null)

const filteredGrafanaDashboards = computed(() => {
  if (!grafanaSearch.value) return grafanaDashboards.value
  const q = grafanaSearch.value.toLowerCase()
  return grafanaDashboards.value.filter(d => d.title.toLowerCase().includes(q) || (d.tags || []).some(t => t.toLowerCase().includes(q)))
})

function onDrop(e) {
  dragging.value = false
  handleFiles(e.dataTransfer.files)
}

function onFileInput(e) {
  handleFiles(e.target.files)
  e.target.value = ''
}

function handleFiles(fileList) {
  error.value = ''
  for (const f of fileList) {
    if (!f.name.endsWith('.json')) {
      error.value = `"${f.name}" is not a JSON file. Export your Grafana dashboard as JSON first.`
      return
    }
    if (!files.value.some(x => x.name === f.name && x.size === f.size)) {
      files.value.push(f)
    }
  }
}

function removeFile(index) {
  files.value.splice(index, 1)
}

async function parse() {
  if (files.value.length === 0) return
  loading.value = true
  error.value = ''
  try {
    const dashboards = []
    for (const f of files.value) {
      const text = await f.text()
      let content
      try { content = JSON.parse(text) } catch { throw new Error(`"${f.name}" is not valid JSON`) }
      dashboards.push({ filename: f.name, content })
    }
    const resp = await fetch('/api/parse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dashboards[0]),
    })
    if (!resp.ok) { const data = await resp.json(); throw new Error(data.error || 'Failed to parse') }
    const parseResult = await resp.json()
    if (parseResult.panels.length === 0) throw new Error('No PromQL panels found in this dashboard.')
    emit('uploaded', { dashboards, parseResult })
  } catch (e) { error.value = e.message }
  finally { loading.value = false }
}

async function connectGrafana() {
  if (!grafanaUrl.value) { error.value = 'Grafana URL is required'; return }
  grafanaLoading.value = true
  error.value = ''
  try {
    const resp = await fetch('/api/grafana/dashboards', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: grafanaUrl.value.replace(/\/+$/, ''), token: grafanaToken.value }),
    })
    if (!resp.ok) { const data = await resp.json(); throw new Error(data.error || 'Connection failed') }
    const data = await resp.json()
    grafanaDashboards.value = data.dashboards || []
    if (grafanaDashboards.value.length === 0) throw new Error('No dashboards found in this Grafana instance.')
  } catch (e) { error.value = e.message }
  finally { grafanaLoading.value = false }
}

async function fetchGrafanaDashboard(d) {
  selectedDashboard.value = d.uid
  loading.value = true
  error.value = ''
  try {
    const resp = await fetch('/api/grafana/dashboard', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify({ url: grafanaUrl.value.replace(/\/+$/, ''), token: grafanaToken.value, uid: d.uid }),
    })
    if (!resp.ok) { const data = await resp.json(); throw new Error(data.error || 'Fetch failed') }
    const data = await resp.json()

    const dashboards = [{ filename: d.title + '.json', content: data.dashboard }]
    const parseResp = await fetch('/api/parse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(dashboards[0]),
    })
    if (!parseResp.ok) { const pd = await parseResp.json(); throw new Error(pd.error || 'Parse failed') }
    const parseResult = await parseResp.json()
    if (parseResult.panels.length === 0) throw new Error('No PromQL panels found in this dashboard.')
    emit('uploaded', { dashboards, parseResult, grafanaUrl: grafanaUrl.value, grafanaToken: grafanaToken.value })
  } catch (e) { error.value = e.message }
  finally { loading.value = false; selectedDashboard.value = null }
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  if (bytes < 1024 * 1024) return (bytes / 1024).toFixed(1) + ' KB'
  return (bytes / 1024 / 1024).toFixed(1) + ' MB'
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-2">Import Dashboard</h2>
      <p class="text-slate-400 leading-relaxed">
        Upload a Grafana JSON file or connect directly to your Grafana instance to pull dashboards.
      </p>
    </div>

    <!-- Mode tabs -->
    <div class="flex bg-slate-800/40 rounded-xl p-1 ring-1 ring-slate-700/40">
      <button
        @click="mode = 'file'"
        :class="['flex-1 flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium transition-all', mode === 'file' ? 'bg-slate-700 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200']"
      >
        <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" /></svg>
        Upload JSON File
      </button>
      <button
        @click="mode = 'grafana'"
        :class="['flex-1 flex items-center justify-center gap-2 py-2.5 rounded-lg text-sm font-medium transition-all', mode === 'grafana' ? 'bg-slate-700 text-white shadow-sm' : 'text-slate-400 hover:text-slate-200']"
      >
        <svg class="w-4 h-4 text-orange-400" viewBox="0 0 24 24" fill="currentColor"><path d="M22.687 12.566c-.045-.498-.18-.907-.315-1.316A8.76 8.76 0 0019.381 7.5C17.5 5.5 15 4.5 12 4.5S6.5 5.5 4.619 7.5A8.76 8.76 0 001.718 11.25c-.135.409-.27.818-.315 1.316C1.358 12.793 1.358 13.019 1.358 13.245s0 .452.045.679c.045.498.18.907.315 1.316A8.76 8.76 0 004.619 18.5C6.5 20.5 9 21.5 12 21.5s5.5-1 7.381-3A8.76 8.76 0 0022.372 15.24c.135-.41.27-.818.315-1.316.045-.227.045-.453.045-.68s0-.452-.045-.679zM12 19.43c-3.555 0-6.43-2.875-6.43-6.43S8.445 6.57 12 6.57s6.43 2.875 6.43 6.43-2.875 6.43-6.43 6.43z"/></svg>
        Connect to Grafana
      </button>
    </div>

    <!-- FILE upload mode -->
    <template v-if="mode === 'file'">
      <div
        @dragenter.prevent="dragging = true"
        @dragover.prevent="dragging = true"
        @dragleave.prevent="dragging = false"
        @drop.prevent="onDrop"
        :class="['relative rounded-2xl border-2 border-dashed p-10 text-center transition-all duration-300 cursor-pointer group', dragging ? 'border-emerald-400 bg-emerald-500/10 scale-[1.01]' : 'border-slate-700/80 hover:border-slate-500 bg-slate-900/30']"
        @click="$refs.fileInput.click()"
      >
        <input ref="fileInput" type="file" accept=".json" multiple class="hidden" @change="onFileInput" />
        <div class="space-y-3">
          <div :class="['mx-auto w-14 h-14 rounded-2xl flex items-center justify-center transition-all duration-300', dragging ? 'bg-emerald-500/20' : 'bg-slate-800 group-hover:bg-slate-700']">
            <svg :class="['w-7 h-7 transition-colors', dragging ? 'text-emerald-400' : 'text-slate-400']" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" /></svg>
          </div>
          <div>
            <p class="text-base font-medium text-slate-200">Drop Grafana JSON files here</p>
            <p class="text-sm text-slate-500 mt-1">or click to browse &middot; Grafana &rarr; Share &rarr; Export &rarr; Save to file</p>
          </div>
        </div>
      </div>

      <div v-if="files.length > 0" class="space-y-2">
        <div v-for="(f, i) in files" :key="f.name + i" class="flex items-center justify-between bg-slate-800/40 rounded-xl px-4 py-3 ring-1 ring-slate-700/40 hover:ring-slate-600/60 transition-all">
          <div class="flex items-center gap-3">
            <div class="w-9 h-9 rounded-lg bg-orange-500/10 flex items-center justify-center">
              <svg class="w-4 h-4 text-orange-400" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
            </div>
            <div>
              <p class="text-sm font-medium text-slate-200">{{ f.name }}</p>
              <p class="text-xs text-slate-500">{{ formatSize(f.size) }}</p>
            </div>
          </div>
          <button @click.stop="removeFile(i)" class="text-slate-600 hover:text-red-400 transition-colors p-1.5 rounded-lg hover:bg-red-500/10">
            <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" /></svg>
          </button>
        </div>
      </div>

      <button @click="parse" :disabled="files.length === 0 || loading" :class="['w-full py-3.5 rounded-xl font-semibold text-sm transition-all duration-200', files.length > 0 && !loading ? 'bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25' : 'bg-slate-800/60 text-slate-600 cursor-not-allowed']">
        <span v-if="loading" class="flex items-center justify-center gap-2">
          <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
          Extracting PromQL panels...
        </span>
        <span v-else>Parse {{ files.length }} dashboard{{ files.length !== 1 ? 's' : '' }}</span>
      </button>
    </template>

    <!-- GRAFANA connection mode -->
    <template v-if="mode === 'grafana'">
      <div class="bg-slate-800/20 rounded-2xl ring-1 ring-slate-700/40 p-5 space-y-4">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-orange-500/15 flex items-center justify-center">
            <svg class="w-5 h-5 text-orange-400" viewBox="0 0 24 24" fill="currentColor"><path d="M22.687 12.566c-.045-.498-.18-.907-.315-1.316A8.76 8.76 0 0019.381 7.5C17.5 5.5 15 4.5 12 4.5S6.5 5.5 4.619 7.5A8.76 8.76 0 001.718 11.25c-.135.409-.27.818-.315 1.316C1.358 12.793 1.358 13.019 1.358 13.245s0 .452.045.679c.045.498.18.907.315 1.316A8.76 8.76 0 004.619 18.5C6.5 20.5 9 21.5 12 21.5s5.5-1 7.381-3A8.76 8.76 0 0022.372 15.24c.135-.41.27-.818.315-1.316.045-.227.045-.453.045-.68s0-.452-.045-.679zM12 19.43c-3.555 0-6.43-2.875-6.43-6.43S8.445 6.57 12 6.57s6.43 2.875 6.43 6.43-2.875 6.43-6.43 6.43z"/></svg>
          </div>
          <div>
            <h3 class="text-sm font-semibold text-slate-200">Grafana Connection</h3>
            <p class="text-xs text-slate-500">Connect to browse and pull dashboards directly</p>
          </div>
        </div>

        <div class="space-y-3">
          <div>
            <label class="block text-xs font-medium text-slate-400 mb-1.5">Grafana URL</label>
            <input v-model="grafanaUrl" type="url" placeholder="https://grafana.example.com" class="w-full bg-slate-900/80 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-orange-500/40 focus:border-orange-500/40 transition" />
          </div>
          <div>
            <label class="block text-xs font-medium text-slate-400 mb-1.5">API Key / Service Account Token <span class="text-slate-600">(optional for public instances)</span></label>
            <input v-model="grafanaToken" type="password" placeholder="glsa_..." class="w-full bg-slate-900/80 border border-slate-700/80 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-orange-500/40 focus:border-orange-500/40 transition" />
            <p class="text-[10px] text-slate-600 mt-1">Grafana &rarr; Administration &rarr; Service accounts &rarr; Add token</p>
          </div>
        </div>

        <button @click="connectGrafana" :disabled="!grafanaUrl || grafanaLoading" :class="['w-full py-2.5 rounded-xl font-semibold text-sm transition-all', grafanaUrl && !grafanaLoading ? 'bg-orange-500 hover:bg-orange-400 text-white shadow-lg shadow-orange-500/20' : 'bg-slate-800/60 text-slate-600 cursor-not-allowed']">
          <span v-if="grafanaLoading" class="flex items-center justify-center gap-2">
            <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
            Connecting...
          </span>
          <span v-else>Connect & List Dashboards</span>
        </button>
      </div>

      <!-- Dashboard browser -->
      <div v-if="grafanaDashboards.length > 0" class="space-y-3">
        <div class="flex items-center justify-between">
          <p class="text-sm font-medium text-slate-300">{{ grafanaDashboards.length }} dashboards found</p>
          <input v-model="grafanaSearch" type="text" placeholder="Filter dashboards..." class="bg-slate-900/60 border border-slate-700/60 rounded-lg px-3 py-1.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-orange-500/30 transition w-48" />
        </div>
        <div class="space-y-1.5 max-h-[45vh] overflow-y-auto pr-1">
          <button
            v-for="d in filteredGrafanaDashboards" :key="d.uid"
            @click="fetchGrafanaDashboard(d)"
            :disabled="loading"
            :class="['w-full text-left rounded-xl ring-1 px-4 py-3 transition-all hover:ring-orange-500/30 group', selectedDashboard === d.uid && loading ? 'bg-orange-500/10 ring-orange-500/30' : 'bg-slate-800/20 ring-slate-700/30 hover:bg-slate-800/40']"
          >
            <div class="flex items-center justify-between">
              <div class="min-w-0 flex-1">
                <p class="text-sm font-medium text-slate-200 truncate group-hover:text-white">{{ d.title }}</p>
                <div v-if="d.tags?.length" class="flex flex-wrap gap-1 mt-1">
                  <span v-for="t in d.tags" :key="t" class="text-[10px] px-1.5 py-0.5 rounded bg-slate-800 text-slate-500 ring-1 ring-slate-700/50">{{ t }}</span>
                </div>
              </div>
              <div v-if="selectedDashboard === d.uid && loading" class="flex-shrink-0 ml-3">
                <svg class="animate-spin w-4 h-4 text-orange-400" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
              </div>
              <svg v-else class="w-4 h-4 text-slate-600 group-hover:text-orange-400 flex-shrink-0 ml-3 transition-colors" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 10v6m0 0l-3-3m3 3l3-3m2 8H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" /></svg>
            </div>
          </button>
          <p v-if="filteredGrafanaDashboards.length === 0" class="text-center text-sm text-slate-600 py-6">No dashboards match your search.</p>
        </div>
      </div>
    </template>

    <!-- Error -->
    <div v-if="error" class="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-red-400 text-sm flex items-start gap-2">
      <svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      {{ error }}
    </div>
  </div>
</template>
