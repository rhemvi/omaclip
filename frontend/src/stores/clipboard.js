import { defineStore } from 'pinia'
import { ref } from 'vue'
import { GetHistory, CopyItem } from '../../wailsjs/go/app/App'

export const useClipboardStore = defineStore('clipboard', () => {
  const items = ref([])
  const selectedIndex = ref(-1)
  const lastCopiedId = ref(null)
  const keyboardActive = ref(false)
  const showShortcuts = ref(false)
  const expandedIds = ref(new Set())

  // fetchHistory loads the current clipboard history from Go.
  async function fetchHistory() {
    items.value = await GetHistory()
  }

// copyItem writes the entry back to the system clipboard then refreshes.
  async function copyItem(id) {
    lastCopiedId.value = id
    await CopyItem(id)
    await fetchHistory()
    setTimeout(() => { lastCopiedId.value = null }, 1000)
  }

  function selectNext() {
    if (items.value.length === 0) return
    keyboardActive.value = true
    selectedIndex.value = Math.min(selectedIndex.value + 1, items.value.length - 1)
  }

  function selectPrev() {
    if (items.value.length === 0) return
    keyboardActive.value = true
    selectedIndex.value = Math.max(selectedIndex.value - 1, 0)
  }

  function clearSelection() {
    selectedIndex.value = -1
    keyboardActive.value = false
  }

  function deactivateKeyboard() {
    keyboardActive.value = false
  }

  function collapseAll() {
    expandedIds.value = new Set()
  }

  function toggleExpanded(id) {
    const s = new Set(expandedIds.value)
    if (s.has(id)) {
      s.delete(id)
    } else {
      s.add(id)
    }
    expandedIds.value = s
  }

  return { items, selectedIndex, lastCopiedId, keyboardActive, showShortcuts, expandedIds, fetchHistory, copyItem, selectNext, selectPrev, clearSelection, deactivateKeyboard, collapseAll, toggleExpanded }
})
