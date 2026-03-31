<script setup>
import { onMounted, onUnmounted } from 'vue'
import { EventsOn, EventsOff } from '../../wailsjs/runtime/runtime'
import { useClipboardStore } from '../stores/clipboard'
import ClipboardItem from './ClipboardItem.vue'

const clipboard = useClipboardStore()
let lastMouseX = 0
let lastMouseY = 0

function handleMouseMove(e) {
  const dx = Math.abs(e.clientX - lastMouseX)
  const dy = Math.abs(e.clientY - lastMouseY)
  if (dx > 3 || dy > 3) {
    clipboard.clearSelection()
  }
  lastMouseX = e.clientX
  lastMouseY = e.clientY
}

function handleKeydown(e) {
  if (e.key === 'ArrowDown') {
    e.preventDefault()
    clipboard.selectNext()
  } else if (e.key === 'ArrowUp') {
    e.preventDefault()
    clipboard.selectPrev()
  } else if (e.key === 'Enter' && clipboard.selectedIndex >= 0) {
    e.preventDefault()
    const item = clipboard.items[clipboard.selectedIndex]
    if (item) {
      clipboard.copyItem(item.id)
      if (clipboard.expandedIds.has(item.id)) clipboard.toggleExpanded(item.id)
    }
  } else if (e.key === ' ' && clipboard.selectedIndex >= 0) {
    e.preventDefault()
    const item = clipboard.items[clipboard.selectedIndex]
    if (item) clipboard.toggleExpanded(item.id)
  } else if (e.key === 'Escape') {
    if (clipboard.expandedIds.size > 0) {
      clipboard.collapseAll()
    } else {
      clipboard.clearSelection()
    }
  } else if (e.ctrlKey && e.key === 'k') {
    e.preventDefault()
    clipboard.showShortcuts = !clipboard.showShortcuts
  } else if (e.ctrlKey && e.key >= '1' && e.key <= '9') {
    e.preventDefault()
    const idx = parseInt(e.key) - 1
    const item = clipboard.items[idx]
    if (item) clipboard.copyItem(item.id)
  }
}

onMounted(() => {
  clipboard.fetchHistory()
  EventsOn('clipboard:new', clipboard.fetchHistory)
  window.addEventListener('keydown', handleKeydown)
  window.addEventListener('mousemove', handleMouseMove)
})

onUnmounted(() => {
  EventsOff('clipboard:new')
  window.removeEventListener('keydown', handleKeydown)
  window.removeEventListener('mousemove', handleMouseMove)
})
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex items-center justify-between px-4 py-3 border-b border-color8">
      <h1 class="text-sm font-semibold text-accent tracking-widest uppercase">Clipboard History</h1>
      <span class="text-[10px] text-color7">Ctrl+K shortcuts</span>
    </div>

    <div class="flex-1 overflow-y-auto">
      <p v-if="clipboard.items.length === 0" class="text-center text-color7 mt-8 text-sm">
        Nothing copied yet.
      </p>
      <ClipboardItem
        v-for="(entry, index) in clipboard.items"
        :key="entry.id"
        :entry="entry"
        :index="index"
        :selected="index === clipboard.selectedIndex"
      />
    </div>
  </div>
</template>
