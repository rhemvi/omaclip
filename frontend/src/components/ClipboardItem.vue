<script setup>
import { ref, watch, onMounted, onUnmounted, nextTick } from 'vue'

const props = defineProps({
  entry: {
    type: Object,
    required: true,
  },
  index: {
    type: Number,
    required: true,
  },
  selected: {
    type: Boolean,
    default: false,
  },
  copied: {
    type: Boolean,
    default: false,
  },
  expanded: {
    type: Boolean,
    default: false,
  },
  keyboardActive: {
    type: Boolean,
    default: false,
  },
})

const emit = defineEmits(['copy', 'toggle-expand'])

const isOverflowing = ref(false)
const hovered = ref(false)
const tooltipStyle = ref({})
const iconRef = ref(null)
const textRef = ref(null)
const rowRef = ref(null)

watch(() => props.selected, (val) => {
  if (val && rowRef.value) {
    rowRef.value.scrollIntoView({ block: 'nearest' })
  }
})

watch(() => props.keyboardActive, (val) => {
  if (val) hovered.value = false
})

function checkOverflow() {
  if (!textRef.value || props.entry.contentType === 'image') return
  isOverflowing.value = textRef.value.scrollWidth > textRef.value.clientWidth
}

onMounted(() => {
  if (props.entry.contentType === 'image') {
    isOverflowing.value = true
  } else {
    nextTick(checkOverflow)
  }
  window.addEventListener('resize', checkOverflow)
})

onUnmounted(() => {
  window.removeEventListener('resize', checkOverflow)
})

function updateTooltipPosition() {
  if (!iconRef.value) return
  const rect = iconRef.value.getBoundingClientRect()
  tooltipStyle.value = {
    position: 'fixed',
    top: `${rect.top - 32}px`,
    right: '8px',
  }
}

function onMouseEnter() {
  hovered.value = true
  updateTooltipPosition()
}

function onMouseLeave() {
  hovered.value = false
}

function handleCopy() {
  emit('copy')
  if (props.expanded) emit('toggle-expand')
}

function formatTime(timestamp) {
  return new Date(timestamp).toLocaleTimeString([], { hour: '2-digit', minute: '2-digit' })
}
</script>

<template>
  <div v-if="entry.contentType === 'image-rejected'" ref="rowRef"
    class="flex items-start gap-3 px-4 py-3 border-b border-color8 opacity-60 transition-colors"
    :class="[selected ? 'bg-color8/50' : '', keyboardActive ? '' : 'hover:bg-color8/50']">
    <span class="shrink-0 w-3 text-[10px] leading-4 text-color7 text-center">!</span>
    <div class="flex-1 min-w-0">
      <p class="text-sm text-color7 italic truncate">{{ entry.content }}</p>
    </div>
    <span class="shrink-0 text-xs text-color7 mt-0.5">{{ formatTime(entry.timestamp) }}</span>
  </div>

  <div v-else ref="rowRef"
    class="group relative flex items-start gap-3 px-4 py-3 border-b border-color8 cursor-pointer transition-colors"
    :class="[selected ? 'bg-color8/50' : '', keyboardActive ? '' : 'hover:bg-color8/50']" @click="handleCopy" @contextmenu.prevent="emit('toggle-expand')"
    @mouseenter="onMouseEnter" @mouseleave="onMouseLeave">
    <span class="shrink-0 w-3 text-[10px] leading-4 text-color2 text-center">{{ index < 9 ? index + 1 : '·' }}</span>

        <div class="flex-1 min-w-0">
          <img v-if="entry.contentType === 'image'"
            :src="'data:' + entry.imageMimeType + ';base64,' + entry.imageData"
            :class="expanded ? 'max-h-64' : 'max-h-16'"
            class="max-w-full rounded object-contain" />
          <p v-else ref="textRef" class="text-sm text-foreground"
            :class="expanded ? 'whitespace-pre-wrap break-all' : 'truncate'">{{ entry.content }}</p>
        </div>

        <button v-if="isOverflowing || expanded"
          class="shrink-0 mt-0.5 text-color6 hover:text-accent cursor-pointer transition-transform"
          :class="expanded ? 'rotate-180' : ''" @click.stop="emit('toggle-expand')">
          <svg xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor" stroke-width="2"
            stroke-linecap="round" stroke-linejoin="round" class="w-4 h-4">
            <polyline points="6 9 12 15 18 9" />
          </svg>
        </button>

        <span class="shrink-0 text-xs text-color7 mt-0.5">{{ formatTime(entry.timestamp) }}</span>

        <div ref="iconRef" class="shrink-0 mt-0.5">
          <span v-if="!copied && selected" class="text-sm text-accent">↵</span>
          <svg v-else-if="!copied" xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"
            stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            :class="hovered ? 'text-accent' : 'text-color6'"
            class="w-4 h-4 transition-colors">
            <rect x="9" y="9" width="13" height="13" rx="2" ry="2" />
            <path d="M5 15H4a2 2 0 0 1-2-2V4a2 2 0 0 1 2-2h9a2 2 0 0 1 2 2v1" />
          </svg>
          <svg v-else xmlns="http://www.w3.org/2000/svg" viewBox="0 0 24 24" fill="none" stroke="currentColor"
            stroke-width="2" stroke-linecap="round" stroke-linejoin="round"
            class="w-4 h-4 text-color2 transition-colors">
            <polyline points="20 6 9 17 4 12" />
          </svg>
        </div>

        <Teleport to="body">
          <span v-if="hovered" :style="tooltipStyle"
            class="pointer-events-none z-50 whitespace-nowrap rounded bg-color0 px-2 py-1 text-xs text-foreground">
            {{ copied ? 'Copied!' : 'Click to copy' }}
          </span>
        </Teleport>
  </div>
</template>
