<script setup lang="ts">
const model = defineModel({
  required: true,
  default: {
    autoStart: false,
    enableGlobalEntrylist: true,
    enableGlobalBanlist: true,
  },
})
</script>

<template>
  <div class="flex flex-col gap-3">
    <UCheckbox v-model="model.autoStart" label="Server instance auto start." />
    <UCheckbox v-model="model.enableGlobalEntrylist" label="Enable global entry list." />
    <UCheckbox v-model="model.enableGlobalBanlist" label="Enable global ban list." />
  </div>
</template>
