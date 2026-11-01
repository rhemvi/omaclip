import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetHistory, CopyItem, GetMaxPinned, SetPinnedIDs } from '../../bindings/github.com/rhemvi/omaclip/app/app.js'
import { useNavigation } from './navigation'

export const useClipboardStore = defineStore('clipboard', () => {
  const items = ref([])
  const pinnedIds = ref([])
  const maxPinned = ref(0)
  const lastCopiedId = ref(null)
  const showShortcuts = ref(false)
  const activeTab = ref('local')

  const orderedItems = computed(() => {
    const byId = new Map(items.value.map((e) => [e.id, e]))
    const pinned = pinnedIds.value.map((id) => byId.get(id)).filter(Boolean)
    const pinnedSet = new Set(pinnedIds.value)
    const unpinned = items.value.filter((e) => !pinnedSet.has(e.id))
    return [...pinned, ...unpinned]
  })

  const nav = useNavigation(() => orderedItems.value)

  function prunePins() {
    const present = new Set(items.value.map((e) => e.id))
    const filtered = pinnedIds.value.filter((id) => present.has(id))
    if (filtered.length !== pinnedIds.value.length) {
      pinnedIds.value = filtered
      syncPinnedToBackend()
    }
  }

  function syncPinnedToBackend() {
    SetPinnedIDs([...pinnedIds.value])
  }

  async function fetchHistory() {
    items.value = await GetHistory()
    prunePins()
  }

  async function fetchMaxPinned() {
    maxPinned.value = await GetMaxPinned()
  }

  function togglePin(id) {
    if (pinnedIds.value.includes(id)) {
      pinnedIds.value = pinnedIds.value.filter((pid) => pid !== id)
      syncPinnedToBackend()
      return
    }
    if (maxPinned.value <= 0) return
    const next = [...pinnedIds.value]
    while (next.length >= maxPinned.value) {
      next.shift()
    }
    next.push(id)
    pinnedIds.value = next
    syncPinnedToBackend()
  }

  async function copyItem(id) {
    lastCopiedId.value = id
    await CopyItem(id)
    await fetchHistory()
    setTimeout(() => { lastCopiedId.value = null }, 1000)
  }

  return {
    items,
    pinnedIds,
    maxPinned,
    orderedItems,
    lastCopiedId,
    showShortcuts,
    activeTab,
    ...nav,
    fetchHistory,
    fetchMaxPinned,
    togglePin,
    copyItem,
  }
})
