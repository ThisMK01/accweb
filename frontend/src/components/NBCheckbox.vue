<script setup lang="ts">
import { ref, watch } from 'vue'

defineProps({
  label: {
    type: String,
    required: true,
  },
  description: {
    type: String,
    default: '',
  },
})

const model = defineModel({ default: 0 })
const boolModel = ref(model.value == 1 ? true : false)

watch(boolModel, (newVal) => {
  model.value = newVal ? 1 : 0
})
</script>

<template>
  <div class="flex items-center gap-2">
    <!-- Componente de Checkbox -->
    <UCheckbox v-bind="$attrs" v-model="boolModel">
      <template #label>
        <div class="flex items-center gap-1.5">
          <!-- Texto da Label -->
          <span>{{ label }}</span>

          <!-- Tooltip com o Ícone -->
          <UTooltip
            :text="description"
            v-if="description"
            portal
            :delay-duration="0"
            :ui="{
              content: 'max-w-100 h-auto py-2 px-3 flex-wrap',
              text: 'whitespace-normal wrap-break-word line-clamp-none',
            }"
          >
            <UIcon name="i-lucide-circle-help" class="w-4 h-4 text-primary cursor-help" />
          </UTooltip>
        </div>
      </template>
    </UCheckbox>
  </div>
</template>
