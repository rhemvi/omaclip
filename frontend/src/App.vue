<script setup>
import { computed, ref, onMounted, onUnmounted } from 'vue'
import { EventsOn } from '../wailsjs/runtime/runtime'
import { GetTheme, NeedsPassphrase, RemoteClipboardsEnabled } from '../wailsjs/go/app/App'
import { useThemeStore } from './stores/theme'
import { useClipboardStore } from './stores/clipboard'
import { useRemoteStore } from './stores/remote'
import ClipboardHistory from './components/ClipboardHistory.vue'
import RemoteClipboard from './components/RemoteClipboard.vue'
import PassphraseSetup from './components/PassphraseSetup.vue'

const themeStore = useThemeStore()
const clipboard = useClipboardStore()
const remote = useRemoteStore()
const needsSetup = ref(false)
const remoteEnabled = ref(false)


const showToast = computed(() => clipboard.lastCopiedId !== null || remote.lastCopiedId !== null)

const allShortcuts = [
  { keys: 'Up / Down', action: 'Navigate items' },
  { keys: 'Enter', action: 'Copy selected item' },
  { keys: 'Space', action: 'Expand / collapse selected' },
  { keys: 'Escape', action: 'Collapse all / clear selection' },
  { keys: 'Ctrl+1..9', action: 'Quick copy Nth item' },
  { keys: ['[', ']'], action: 'Navigate tabs', remoteOnly: true },
  { keys: 'Ctrl+K', action: 'Toggle this panel' },
]
const shortcuts = computed(() => allShortcuts.filter(s => !s.remoteOnly || remoteEnabled.value))

let lastMouseX = 0
let lastMouseY = 0

function handleMouseMove(e) {
  const dx = Math.abs(e.clientX - lastMouseX)
  const dy = Math.abs(e.clientY - lastMouseY)
  if (dx > 3 || dy > 3) {
    const store = clipboard.activeTab === 'local' ? clipboard : remote
    store.clearSelection()
  }
  lastMouseX = e.clientX
  lastMouseY = e.clientY
}

function handleKeydown(e) {
  if (e.key === '[' && remoteEnabled.value) {
    e.preventDefault()
    clipboard.activeTab = 'local'
    return
  } else if (e.key === ']' && remoteEnabled.value) {
    e.preventDefault()
    clipboard.activeTab = 'remote'
    return
  }

  const store = clipboard.activeTab === 'local' ? clipboard : remote
  const items = clipboard.activeTab === 'local' ? clipboard.items : remote.flatEntries

  if (e.key === 'ArrowDown') {
    e.preventDefault()
    store.selectNext()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    store.selectPrev()
  } else if (e.key === 'Enter' && store.selectedIndex >= 0) {
    e.preventDefault()
    const item = items[store.selectedIndex]
    if (item) {
      store.copyItem(item.id)
      if (store.expandedIds.has(item.id)) store.toggleExpanded(item.id)
    }
  } else if (e.key === ' ' && store.selectedIndex >= 0) {
    e.preventDefault()
    const item = items[store.selectedIndex]
    if (item) store.toggleExpanded(item.id)
  } else if (e.key === 'Escape') {
    if (store.expandedIds.size > 0) {
      store.collapseAll()
    } else {
      store.clearSelection()
    }
  } else if (e.ctrlKey && e.key === 'k') {
    e.preventDefault()
    clipboard.showShortcuts = !clipboard.showShortcuts
  } else if (e.ctrlKey && e.key >= '1' && e.key <= '9') {
    e.preventDefault()
    const idx = parseInt(e.key) - 1
    const item = items[idx]
    if (item) store.copyItem(item.id)
  }
}

onMounted(async () => {
  EventsOn('theme:loaded', themeStore.applyColors)

  const colors = await GetTheme()
  if (colors && colors.background) {
    themeStore.applyColors(colors)
  }

  remoteEnabled.value = await RemoteClipboardsEnabled()
  needsSetup.value = await NeedsPassphrase()

  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('mousemove', handleMouseMove)
})

onUnmounted(() => {
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('mousemove', handleMouseMove)
})
</script>

<template>
  <PassphraseSetup v-if="needsSetup" @done="needsSetup = false" />
  <div v-else class="min-h-screen bg-background text-foreground font-mono flex flex-col">
    <div class="flex border-b border-color8">
      <button
        class="px-4 py-2.5 text-xs font-semibold tracking-widest uppercase transition-colors border-b-2 cursor-pointer"
        :class="clipboard.activeTab === 'local' ? 'text-accent border-accent' : 'text-color7 border-transparent hover:text-foreground'"
        @click="clipboard.activeTab = 'local'"
        @mousedown.prevent
      >
        Clipboard
      </button>
      <button
        v-if="remoteEnabled"
        class="px-4 py-2.5 text-xs font-semibold tracking-widest uppercase transition-colors border-b-2 cursor-pointer"
        :class="clipboard.activeTab === 'remote' ? 'text-accent border-accent' : 'text-color7 border-transparent hover:text-foreground'"
        @click="clipboard.activeTab = 'remote'"
        @mousedown.prevent
      >
        Remote Clipboard{{ remote.peers.length > 1 ? 's' : '' }}
        <span v-if="remote.peers.length > 0" class="ml-1.5 text-[10px] text-color7 normal-case tracking-normal">{{ remote.peers.length }}</span>
      </button>
      <div class="ml-auto flex items-center pr-4">
        <span class="text-[10px] text-color7">Ctrl+K shortcuts</span>
      </div>
    </div>

    <div class="flex-1 overflow-hidden">
      <ClipboardHistory v-show="clipboard.activeTab === 'local'" />
      <RemoteClipboard v-if="remoteEnabled" v-show="clipboard.activeTab === 'remote'" />
    </div>

    <Transition
      enter-active-class="transition-opacity duration-200"
      leave-active-class="transition-opacity duration-300"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="showToast"
        class="fixed bottom-4 left-1/2 -translate-x-1/2 rounded bg-color2/90 px-3 py-1.5 text-xs font-medium text-background"
      >
        Copied to clipboard
      </div>
    </Transition>

    <Transition
      enter-active-class="transition-opacity duration-150"
      leave-active-class="transition-opacity duration-150"
      enter-from-class="opacity-0"
      leave-to-class="opacity-0"
    >
      <div
        v-if="clipboard.showShortcuts"
        class="fixed inset-0 z-50 flex items-center justify-center bg-background/80"
        @click.self="clipboard.showShortcuts = false"
      >
        <div class="w-72 rounded-lg border border-color8 bg-color0 p-4">
          <h2 class="text-sm font-semibold text-accent mb-3">Keyboard Shortcuts</h2>
          <div class="space-y-2">
            <div v-for="s in shortcuts" :key="s.keys" class="flex items-center justify-between">
              <span class="text-xs text-foreground">{{ s.action }}</span>
              <span class="flex gap-1">
                <template v-if="Array.isArray(s.keys)">
                  <kbd v-for="k in s.keys" :key="k" class="rounded bg-color8 px-1.5 py-0.5 text-[10px] text-color7">{{ k }}</kbd>
                </template>
                <kbd v-else class="rounded bg-color8 px-1.5 py-0.5 text-[10px] text-color7">{{ s.keys }}</kbd>
              </span>
            </div>
          </div>
          <p class="mt-3 text-center text-[10px] text-color7">Press Ctrl+K or click outside to close</p>
        </div>
      </div>
    </Transition>
  </div>
</template>
