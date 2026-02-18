<script setup>
import { ref, computed } from 'vue'
import StepUpload from './components/StepUpload.vue'
import StepConfig from './components/StepConfig.vue'
import StepResults from './components/StepResults.vue'
import StepOutput from './components/StepOutput.vue'

const step = ref(1)
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

const steps = [
  { num: 1, label: 'Upload' },
  { num: 2, label: 'Configure' },
  { num: 3, label: 'Results' },
  { num: 4, label: 'Export' },
]

function onUploaded(data) {
  dashboards.value = data.dashboards
  parseResult.value = data.parseResult
  if (data.parseResult?.title) {
    config.value.name = data.parseResult.title
  }
  step.value = 2
}

function onConfigured(cfg) {
  Object.assign(config.value, cfg)
  step.value = 3
}

function onChecked(result) {
  checkResult.value = result
}

function onConvert() {
  step.value = 4
}

function onConverted(result) {
  convertResult.value = result
}

function reset() {
  step.value = 1
  dashboards.value = []
  parseResult.value = null
  checkResult.value = null
  convertResult.value = null
  config.value = {
    stsUrl: '',
    stsToken: '',
    name: '',
    metricPrefix: '',
    rewriteMetrics: true,
    includeMissing: false,
  }
}
</script>

<template>
  <div class="min-h-screen bg-gradient-to-br from-slate-950 via-slate-900 to-slate-950 text-white">
    <header class="border-b border-slate-800/60 backdrop-blur-sm bg-slate-900/50 sticky top-0 z-50">
      <div class="max-w-6xl mx-auto px-6 py-4 flex items-center justify-between">
        <div class="flex items-center gap-3">
          <div class="w-9 h-9 rounded-lg bg-gradient-to-br from-emerald-400 to-teal-500 flex items-center justify-center font-bold text-slate-900 text-lg">O</div>
          <div>
            <h1 class="text-xl font-bold tracking-tight">Odyssey</h1>
            <p class="text-xs text-slate-400">Grafana → SUSE Observability</p>
          </div>
        </div>
        <button v-if="step > 1" @click="reset" class="text-sm text-slate-400 hover:text-white transition-colors px-3 py-1.5 rounded-lg hover:bg-slate-800">
          Start over
        </button>
      </div>
    </header>

    <nav class="max-w-6xl mx-auto px-6 pt-8 pb-2">
      <div class="flex items-center gap-2">
        <template v-for="(s, i) in steps" :key="s.num">
          <div
            class="flex items-center gap-2 px-3 py-1.5 rounded-full text-sm font-medium transition-all duration-300"
            :class="{
              'bg-emerald-500/20 text-emerald-400 ring-1 ring-emerald-500/30': step === s.num,
              'text-slate-400': step !== s.num && step < s.num,
              'text-slate-300': step !== s.num && step > s.num,
            }"
          >
            <span
              class="w-6 h-6 rounded-full flex items-center justify-center text-xs font-bold transition-all"
              :class="{
                'bg-emerald-500 text-slate-900': step >= s.num,
                'bg-slate-700 text-slate-400': step < s.num,
              }"
            >
              <template v-if="step > s.num">&#10003;</template>
              <template v-else>{{ s.num }}</template>
            </span>
            {{ s.label }}
          </div>
          <div v-if="i < steps.length - 1" class="flex-1 h-px bg-slate-800" />
        </template>
      </div>
    </nav>

    <main class="max-w-6xl mx-auto px-6 py-8">
      <Transition name="fade" mode="out-in">
        <StepUpload v-if="step === 1" @uploaded="onUploaded" />
        <StepConfig
          v-else-if="step === 2"
          :parse-result="parseResult"
          :config="config"
          @configured="onConfigured"
          @back="step = 1"
        />
        <StepResults
          v-else-if="step === 3"
          :dashboards="dashboards"
          :config="config"
          @checked="onChecked"
          @convert="onConvert"
          @back="step = 2"
        />
        <StepOutput
          v-else-if="step === 4"
          :dashboards="dashboards"
          :config="config"
          :check-result="checkResult"
          @converted="onConverted"
          @back="step = 3"
        />
      </Transition>
    </main>

    <footer class="border-t border-slate-800/60 mt-auto">
      <div class="max-w-6xl mx-auto px-6 py-4 text-center text-xs text-slate-500">
        Odyssey &mdash; Open source, Apache 2.0
      </div>
    </footer>
  </div>
</template>

<style>
.fade-enter-active, .fade-leave-active {
  transition: opacity 0.2s ease, transform 0.2s ease;
}
.fade-enter-from {
  opacity: 0;
  transform: translateY(8px);
}
.fade-leave-to {
  opacity: 0;
  transform: translateY(-8px);
}
</style>
