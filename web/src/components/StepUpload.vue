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
}

function handleFiles(fileList) {
  error.value = ''
  const newFiles = []
  for (const f of fileList) {
    if (!f.name.endsWith('.json')) {
      error.value = `${f.name} is not a JSON file`
      return
    }
    newFiles.push(f)
  }
  files.value = [...files.value, ...newFiles]
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
      const content = JSON.parse(text)
      dashboards.push({ filename: f.name, content })
    }

    const first = dashboards[0]
    const resp = await fetch('/api/parse', {
      method: 'POST',
      headers: { 'Content-Type': 'application/json' },
      body: JSON.stringify(first),
    })

    if (!resp.ok) {
      const data = await resp.json()
      throw new Error(data.error || 'Parse failed')
    }

    const parseResult = await resp.json()
    emit('uploaded', { dashboards, parseResult })
  } catch (e) {
    error.value = e.message
  } finally {
    loading.value = false
  }
}

function formatSize(bytes) {
  if (bytes < 1024) return bytes + ' B'
  return (bytes / 1024).toFixed(1) + ' KB'
}
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-1">Upload Grafana Dashboard</h2>
      <p class="text-slate-400">Drop your Grafana JSON export file(s) below to begin the conversion.</p>
    </div>

    <div
      @dragenter.prevent="dragging = true"
      @dragover.prevent="dragging = true"
      @dragleave.prevent="dragging = false"
      @drop.prevent="onDrop"
      :class="[
        'relative rounded-2xl border-2 border-dashed p-12 text-center transition-all duration-200 cursor-pointer',
        dragging
          ? 'border-emerald-400 bg-emerald-500/10'
          : 'border-slate-700 hover:border-slate-500 bg-slate-900/50',
      ]"
      @click="$refs.fileInput.click()"
    >
      <input ref="fileInput" type="file" accept=".json" multiple class="hidden" @change="onFileInput" />
      <div class="space-y-3">
        <div class="mx-auto w-16 h-16 rounded-2xl bg-slate-800 flex items-center justify-center">
          <svg class="w-8 h-8 text-slate-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="1.5" d="M7 16a4 4 0 01-.88-7.903A5 5 0 1115.9 6L16 6a5 5 0 011 9.9M15 13l-3-3m0 0l-3 3m3-3v12" />
          </svg>
        </div>
        <div>
          <p class="text-lg font-medium text-slate-200">Drop Grafana JSON files here</p>
          <p class="text-sm text-slate-500 mt-1">or click to browse &middot; .json files only</p>
        </div>
      </div>
    </div>

    <div v-if="files.length > 0" class="space-y-2">
      <div
        v-for="(f, i) in files"
        :key="f.name + i"
        class="flex items-center justify-between bg-slate-800/60 rounded-xl px-4 py-3 ring-1 ring-slate-700/50"
      >
        <div class="flex items-center gap-3">
          <div class="w-8 h-8 rounded-lg bg-emerald-500/15 flex items-center justify-center">
            <svg class="w-4 h-4 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
              <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12h6m-6 4h6m2 5H7a2 2 0 01-2-2V5a2 2 0 012-2h5.586a1 1 0 01.707.293l5.414 5.414a1 1 0 01.293.707V19a2 2 0 01-2 2z" />
            </svg>
          </div>
          <div>
            <p class="text-sm font-medium text-slate-200">{{ f.name }}</p>
            <p class="text-xs text-slate-500">{{ formatSize(f.size) }}</p>
          </div>
        </div>
        <button @click.stop="removeFile(i)" class="text-slate-500 hover:text-red-400 transition-colors p-1">
          <svg class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M6 18L18 6M6 6l12 12" />
          </svg>
        </button>
      </div>
    </div>

    <div v-if="error" class="bg-red-500/10 border border-red-500/30 rounded-xl px-4 py-3 text-red-400 text-sm">
      {{ error }}
    </div>

    <button
      @click="parse"
      :disabled="files.length === 0 || loading"
      :class="[
        'w-full py-3 rounded-xl font-semibold text-sm transition-all duration-200',
        files.length > 0 && !loading
          ? 'bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25'
          : 'bg-slate-800 text-slate-500 cursor-not-allowed',
      ]"
    >
      <span v-if="loading" class="flex items-center justify-center gap-2">
        <svg class="animate-spin w-4 h-4" fill="none" viewBox="0 0 24 24">
          <circle class="opacity-25" cx="12" cy="12" r="10" stroke="currentColor" stroke-width="4" />
          <path class="opacity-75" fill="currentColor" d="M4 12a8 8 0 018-8V0C5.373 0 0 5.373 0 12h4z" />
        </svg>
        Parsing...
      </span>
      <span v-else>Parse {{ files.length }} dashboard{{ files.length !== 1 ? 's' : '' }}</span>
    </button>
  </div>
</template>
