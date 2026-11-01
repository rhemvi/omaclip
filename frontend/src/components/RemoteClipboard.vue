<script setup>
import { onMounted, onUnmounted } from 'vue'
import { Events } from '@wailsio/runtime'
import { useRemoteStore } from '../stores/remote'
import ClipboardItem from './ClipboardItem.vue'

const remote = useRemoteStore()
let unsubscribeRemote

function entriesForPeer(peerName) {
  return remote.flatEntries.filter(e => e.peerName === peerName)
}

onMounted(() => {
  remote.fetchRemote()
  unsubscribeRemote = Events.On('remote:updated', () => remote.fetchRemote())
})

onUnmounted(() => {
  unsubscribeRemote?.()
})
</script>

<template>
  <div class="flex flex-col h-full">
    <div class="flex-1 overflow-y-auto">
      <p v-if="remote.peers.length === 0" class="text-center text-color7 mt-8 text-sm">
        No remote peers found.
      </p>
      <template v-for="peer in remote.peers" :key="peer.peerName">
        <div class="px-4 py-1.5 bg-color0/50 border-b border-color8">
          <span class="text-xs text-color4 font-medium">{{ peer.peerName }}</span>
        </div>
        <p v-if="entriesForPeer(peer.peerName).length === 0" class="text-center text-color7 py-4 text-sm">
          No content yet — copy something on the remote to see it here.
        </p>
        <ClipboardItem
          v-for="entry in entriesForPeer(peer.peerName)"
          :key="entry.id"
          :entry="entry"
          :pinnable="false"
          :index="remote.flatEntries.indexOf(entry)"
          :selected="remote.flatEntries.indexOf(entry) === remote.selectedIndex"
          :copied="remote.lastCopiedId === entry.id"
          :expanded="remote.expandedIds.has(entry.id)"
          :keyboard-active="remote.keyboardActive"
          @copy="remote.copyItem(entry.id)"
          @toggle-expand="remote.toggleExpanded(entry.id)"
        />
      </template>
    </div>
  </div>
</template>
