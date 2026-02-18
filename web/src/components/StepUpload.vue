<script setup>
import { ref } from 'vue'

const emit = defineEmits(['uploaded'])

const files = ref([])
const dragging = ref(false)
const error = ref('')
const loading = ref(false)

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

    if (!resp.ok) {
      const data = await resp.json()
      throw new Error(data.error || 'Failed to parse dashboard')
    }

    const parseResult = await resp.json()
    if (parseResult.panels.length === 0) {
      throw new Error('No PromQL panels found in this dashboard. Make sure it contains Prometheus-based panels.')
    }
    emit('uploaded', { dashboards, parseResult })
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
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
      <h2 class="text-2xl font-bold mb-2">Upload Grafana Dashboard</h2>
      <p class="text-slate-400 leading-relaxed">
        Export your dashboard from Grafana (<span class="text-slate-300">Dashboard &rarr; Share &rarr; Export &rarr; Save to file</span>) and drop the JSON file below.
        Odyssey will extract all PromQL queries from every panel.
      </p>
    </div>

    <!-- Drop zone -->
    <div
      @dragenter.prevent="dragging = true"
      @dragover.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @drop.prevent="onDrop"
      :class="[
        'relative rounded-2xl border-2 border-dashed p-10 text-center transition-all duration-300 cursor-pointer group',
        dragging
          ? 'border-emerald-400 bg-emerald-500/10 scale-[1.01]'
          : 'border-slate-700/80 hover:border-slate-500 bg-slate-900/30',
      ]"
      @click="$refs.fileInput.click()"
    >
      <input ref="fileInput" type="file" accept=".json" multiple class="hidden" @change="onFileInput" />
      <div class="space-y-3">
        <div :class="['mx-auto w-14 h-14 rounded-2xl flex items-center justify-center transition-all duration-300', dragging ? 'bg-emerald-500/20' : 'bg-slate-800 group-hover:bg-slate-700']">
          <svg :class="['w-7 h-7 transition-colors', dragging ? 'text-emerald-400' : 'text-slate-400']" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>
        </div>
        <div>
          <p class="text-base font-medium text-slate-200">Drop Grafana JSON files here</p>
          <p class="text-sm text-slate-500 mt-1">or click to browse &middot; .json only &middot; multiple files supported</p>
        </div>
      </div>
    </div>

    <!-- File list -->
    <div v-if="files.length > 0" class="space-y-2">
      <p class="text-xs text-slate-500 font-medium uppercase tracking-wider">{{ files.length }} file{{ files.length !== 1 ? 's' : '' }} selected</p>
      <div
        v-for="(f, i) in files"
        :key="f.name + i"
        class="flex items-center justify-between bg-slate-800/40 rounded-xl px-4 py-3 ring-1 ring-slate-700/40 group hover:ring-slate-600/60 transition-all"
      >
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-orange-500/10 flex items-center justify-center">
            <svg class="w-4.5 h-4.5 text-orange-400" viewBox="0 0 24 24" fill="currentColor"><path d="M22.687 12.566c-.045-.498-.18-.907-.315-1.316-.045-.136-.09-.317-.135-.453a7.5 7.5 0 00-.27-.725A8.76 8.76 0 0019.381 7.5C17.5 5.5 15 4.5 12 4.5S6.5 5.5 4.619 7.5A8.76 8.76 0 002.033 10.072c-.135.409-.27.818-.315 1.316C1.673 11.614 1.673 11.841 1.673 12.067s0 .452.045.679c.045.498.18.907.315 1.316A8.76 8.76 0 004.619 16.5C6.5 18.5 9 19.5 12 19.5s5.5-1 7.381-3A8.76 8.76 0 0021.967 13.928c.135-.41.27-.818.315-1.316.045-.227.045-.453.045-.68s0-.452-.045-.679zM12 18.43c-3.555 0-6.43-2.875-6.43-6.43S8.445 5.57 12 5.57s6.43 2.875 6.43 6.43-2.875 6.43-6.43 6.43z"/></svg>
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

    <!-- Error -->
    <div v-if="error" class="bg-red-500/10 border border-red-500/20 rounded-xl px-4 py-3 text-red-400 text-sm flex items-start gap-2">
      <svg class="w-4 h-4 mt-0.5 flex-shrink-0" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M12 8v4m0 4h.01M21 12a9 9 0 11-18 0 9 9 0 0118 0z" /></svg>
      {{ error }}
    </div>

    <!-- Parse button -->
    <button
      @click="parse"
      :disabled="files.length === 0 || loading"
      :class="[
        'w-full py-3.5 rounded-xl font-semibold text-sm transition-all duration-200',
        files.length > 0 && !loading
          ? 'bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25'
          : 'bg-slate-800/60 text-slate-600 cursor-not-allowed',
      ]"
    >
      <span v-if="loading" class="flex items-center justify-center gap-2">
        <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24"><circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" /><path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" /></svg>
        Extracting PromQL panels...
      </span>
      <span v-else>Parse {{ files.length }} dashboard{{ files.length !== 1 ? 's' : '' }}</span>
    </button>

    <!-- Help text -->
    <div v-if="files.length === 0" class="text-center space-y-2">
      <p class="text-xs text-slate-600">Supported: PostgreSQL, MySQL, NGINX, Kubernetes, Node Exporter, and 40+ more</p>
    </div>
  </div>
</template>
