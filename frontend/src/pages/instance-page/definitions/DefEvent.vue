<script setup lang="ts">
import SessionsTable from '@/components/SessionsTable.vue';
import { type EventJson } from '@/lib/accweb/types'

const model = defineModel({
  required: true,
  default: {} as EventJson,
})
</script>

<template>
  <div class="flex flex-row gap-2">
    <div class="flex-1">
      {{ model.sessions }}
    </div>
    <div class="flex-1">
      <SessionsTable v-model="model.sessions" />
    </div>
  </div>
</template>
