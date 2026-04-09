import { defineStore } from 'pinia'
import { ref, computed } from 'vue'
import { GetRemoteClipboards, CopyRemoteItem, CopyRemoteImage } from '../../wailsjs/go/app/App'
import { useNavigation } from './navigation'

export const useRemoteStore = defineStore('remote', () => {
  const peers = ref([])
  const lastCopiedId = ref(null)

  const flatEntries = computed(() => {
    const result = []
    for (const peer of peers.value) {
      for (const entry of (peer.entries || [])) {
        result.push({ ...entry, peerName: peer.peerName })
      }
    }
    return result
  })

  const nav = useNavigation(() => flatEntries.value)

  async function fetchRemote() {
    const result = await GetRemoteClipboards()
    peers.value = result || []
  }

  async function copyItem(id) {
    const entry = flatEntries.value.find(e => e.id === id)
    if (!entry) return
    lastCopiedId.value = id
    if (entry.contentType === 'image') {
      await CopyRemoteImage(entry.imageData, entry.imageMimeType)
    } else {
      await CopyRemoteItem(entry.content)
    }
    setTimeout(() => { lastCopiedId.value = null }, 1000)
  }

  return { peers, flatEntries, lastCopiedId, ...nav, fetchRemote, copyItem }
})
