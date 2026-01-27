<script setup lang="ts">
import EntryList from '@/components/EntryList.vue'
import type { EntrylistJson } from '@/lib/accweb/types'

const model = defineModel({
  required: true,
  default: {} as EntrylistJson,
})
</script>

<template>
  <EntryList v-model="model" />
</template>
