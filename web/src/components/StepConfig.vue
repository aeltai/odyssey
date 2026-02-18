<script setup>
import { ref } from 'vue'

const props = defineProps(['parseResult', 'config'])
const emit = defineEmits(['configured', 'back'])

const stsUrl = ref(props.config.stsUrl || '')
const stsToken = ref(props.config.stsToken || '')
const name = ref(props.config.name || props.parseResult?.title || '')
const metricPrefix = ref(props.config.metricPrefix || '')
const rewriteMetrics = ref(props.config.rewriteMetrics ?? true)
const includeMissing = ref(props.config.includeMissing ?? false)
const showToken = ref(false)

function proceed() {
  emit('configured', {
    stsUrl: stsUrl.value.replace(/\/+$/, ''),
    stsToken: stsToken.value,
    name: name.value,
    metricPrefix: metricPrefix.value,
    rewriteMetrics: rewriteMetrics.value,
    includeMissing: includeMissing.value,
  })
}

const panelCount = props.parseResult?.panels?.length || 0
const warnings = props.parseResult?.panels?.filter(p => p.warning)?.length || 0
</script>

<template>
  <div class="space-y-6">
    <div>
      <h2 class="text-2xl font-bold mb-1">Configure</h2>
      <p class="text-slate-400">Connect to SUSE Observability and set conversion options.</p>
    </div>

    <div class="bg-slate-800/40 rounded-2xl ring-1 ring-slate-700/50 p-5 space-y-1">
      <div class="flex items-center gap-3">
        <div class="w-10 h-10 rounded-xl bg-emerald-500/15 flex items-center justify-center">
          <svg class="w-5 h-5 text-emerald-400" fill="none" stroke="currentColor" viewBox="0 0 24 24">
            <path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M9 12l2 2 4-4m6 2a9 9 0 11-18 0 9 9 0 0118 0z" />
          </svg>
        </div>
        <div>
          <p class="font-semibold text-slate-200">{{ name || 'Dashboard' }}</p>
          <p class="text-sm text-slate-400">
            {{ panelCount }} panel{{ panelCount !== 1 ? 's' : '' }} parsed
            <span v-if="warnings" class="text-amber-400">&middot; {{ warnings }} warning{{ warnings !== 1 ? 's' : '' }}</span>
          </p>
        </div>
      </div>
    </div>

    <div class="space-y-5">
      <div>
        <h3 class="text-sm font-semibold text-slate-300 uppercase tracking-wider mb-3">SUSE Observability Connection</h3>
        <p class="text-xs text-slate-500 mb-3">Optional. Connect to check metric availability and auto-detect prefixes.</p>
        <div class="space-y-3">
          <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">URL</label>
            <input
              v-model="stsUrl"
              type="url"
              placeholder="https://observability.example.com"
              class="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">API Token</label>
            <div class="relative">
              <input
                v-model="stsToken"
                :type="showToken ? 'text' : 'password'"
                placeholder="your-api-token"
                class="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-2.5 pr-12 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition"
              />
              <button
                @click="showToken = !showToken"
                class="absolute right-3 top-1/2 -translate-y-1/2 text-slate-500 hover:text-slate-300 transition"
              >
                <svg v-if="!showToken" class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M15 12a3 3 0 11-6 0 3 3 0 016 0z" /><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M2.458 12C3.732 7.943 7.523 5 12 5c4.478 0 8.268 2.943 9.542 7-1.274 4.057-5.064 7-9.542 7-4.477 0-8.268-2.943-9.542-7z" /></svg>
                <svg v-else class="w-4 h-4" fill="none" stroke="currentColor" viewBox="0 0 24 24"><path stroke-linecap="round" stroke-linejoin="round" stroke-width="2" d="M13.875 18.825A10.05 10.05 0 0112 19c-4.478 0-8.268-2.943-9.543-7a9.97 9.97 0 011.563-3.029m5.858.908a3 3 0 114.243 4.243M9.878 9.878l4.242 4.242M9.88 9.88l-3.29-3.29m7.532 7.532l3.29 3.29M3 3l3.59 3.59m0 0A9.953 9.953 0 0112 5c4.478 0 8.268 2.943 9.543 7a10.025 10.025 0 01-4.132 5.411m0 0L21 21" /></svg>
              </button>
            </div>
          </div>
        </div>
      </div>

      <hr class="border-slate-800" />

      <div>
        <h3 class="text-sm font-semibold text-slate-300 uppercase tracking-wider mb-3">Conversion Options</h3>
        <div class="space-y-3">
          <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">Dashboard Name</label>
            <input
              v-model="name"
              type="text"
              placeholder="My Dashboard"
              class="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition"
            />
          </div>
          <div>
            <label class="block text-sm font-medium text-slate-400 mb-1">Metric Prefix <span class="text-slate-500">(auto-detected if empty)</span></label>
            <input
              v-model="metricPrefix"
              type="text"
              placeholder="e.g. postgresql"
              class="w-full bg-slate-900 border border-slate-700 rounded-xl px-4 py-2.5 text-sm text-slate-200 placeholder-slate-600 focus:outline-none focus:ring-2 focus:ring-emerald-500/40 focus:border-emerald-500/40 transition"
            />
          </div>
          <label class="flex items-center gap-3 cursor-pointer group">
            <div
              :class="[
                'w-10 h-6 rounded-full relative transition-colors duration-200',
                rewriteMetrics ? 'bg-emerald-500' : 'bg-slate-700',
              ]"
              @click="rewriteMetrics = !rewriteMetrics"
            >
              <div
                :class="[
                  'w-4 h-4 rounded-full bg-white absolute top-1 transition-transform duration-200',
                  rewriteMetrics ? 'translate-x-5' : 'translate-x-1',
                ]"
              />
            </div>
            <span class="text-sm text-slate-300">Rewrite metric names with prefix</span>
          </label>
          <label class="flex items-center gap-3 cursor-pointer group">
            <div
              :class="[
                'w-10 h-6 rounded-full relative transition-colors duration-200',
                includeMissing ? 'bg-emerald-500' : 'bg-slate-700',
              ]"
              @click="includeMissing = !includeMissing"
            >
              <div
                :class="[
                  'w-4 h-4 rounded-full bg-white absolute top-1 transition-transform duration-200',
                  includeMissing ? 'translate-x-5' : 'translate-x-1',
                ]"
              />
            </div>
            <span class="text-sm text-slate-300">Include panels with missing metrics</span>
          </label>
        </div>
      </div>
    </div>

    <div class="flex gap-3">
      <button
        @click="$emit('back')"
        class="px-6 py-3 rounded-xl text-sm font-medium text-slate-400 hover:text-white bg-slate-800 hover:bg-slate-700 transition-all"
      >
        Back
      </button>
      <button
        @click="proceed"
        class="flex-1 py-3 rounded-xl font-semibold text-sm bg-emerald-500 hover:bg-emerald-400 text-slate-900 shadow-lg shadow-emerald-500/25 transition-all"
      >
        {{ stsUrl && stsToken ? 'Check Metrics & Continue' : 'Continue without metric check' }}
      </button>
    </div>
  </div>
</template>
