<script setup>
import { ref, onMounted } from 'vue'
import { SubmitPassphrase, GetConfigPath } from '../../bindings/github.com/rhemvi/omaclip/app/app.js'

const emit = defineEmits(['done'])
const passphrase = ref('')
const error = ref('')
const configPath = ref('')

onMounted(async () => {
  configPath.value = await GetConfigPath()
})

function validate(value) {
  if (value.length < 8) return 'Passphrase must be at least 8 characters'
  if (value.length > 128) return 'Passphrase must be at most 128 characters'
  if (value !== value.trimStart()) return 'Passphrase must not start with whitespace'
  if (value !== value.trimEnd()) return 'Passphrase must not end with whitespace'
  return ''
}

async function submit() {
  error.value = validate(passphrase.value)
  if (error.value) return
  try {
    await SubmitPassphrase(passphrase.value)
    emit('done')
  } catch (e) {
    error.value = e
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center bg-background">
    <div class="w-80 rounded-lg border border-color8 bg-color0 p-6">
      <h1 class="text-lg font-bold text-accent mb-1">Omaclip</h1>
      <p class="text-xs text-color7 mb-6">
        Enter a passphrase to share clipboards with other machines. Only devices with the same passphrase will sync.
      </p>

      <label class="block text-xs text-foreground mb-1.5">Passphrase</label>
      <input v-model="passphrase" type="password" placeholder="shared secret" autofocus
        class="w-full rounded border border-color8 bg-background px-3 py-2 text-sm text-foreground placeholder-color7 focus:border-accent focus:outline-none"
        @keydown.enter="submit" />
      <p v-if="error" class="mt-1 text-[11px] text-color1">{{ error }}</p>

      <button
        class="mt-4 w-full rounded bg-accent px-3 py-2 text-sm font-semibold text-background hover:opacity-90 transition-opacity"
        @click="submit">
        Continue
      </button>

      <p class="mt-3 text-center text-[10px] text-color7">
        Stored locally in {{ configPath }}
      </p>
    </div>
  </div>
</template>
