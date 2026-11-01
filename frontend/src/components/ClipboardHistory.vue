<script setup>
import { onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import { useClipboardStore } from '../stores/clipboard'
import ClipboardItem from './ClipboardItem.vue'

const clipboard = useClipboardStore()
let unsubscribeClipboard

onMounted(() => {
  clipboard.fetchMaxPinned()
  clipboard.fetchHistory()
  unsubscribeClipboard = Events.On('clipboard:new', () => clipboard.fetchHistory())
})

onUnmounted(() => {
  unsubscribeClipboard?.()
})
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto">
      <p v-if="clipboard.orderedItems.length === 0" class="text-center text-color7 mt-8 text-sm">
        Nothing copied yet.
      </p>
      <ClipboardItem
        v-for="(entry, index) in clipboard.orderedItems"
        :key="entry.id"
        :entry="entry"
        :index="index"
        :selected="index === clipboard.selectedIndex"
        :copied="clipboard.lastCopiedId === entry.id"
        :expanded="clipboard.expandedIds.has(entry.id)"
        :pinned="clipboard.pinnedIds.includes(entry.id)"
        :keyboard-active="clipboard.keyboardActive"
        @copy="clipboard.copyItem(entry.id)"
        @toggle-expand="clipboard.toggleExpanded(entry.id)"
        @toggle-pin="clipboard.togglePin(entry.id)"
      />
    </div>
  </div>
</template>
