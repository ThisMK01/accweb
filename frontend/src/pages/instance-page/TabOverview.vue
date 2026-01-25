<script lang="ts" setup>
import type { LiveServerInstancePayload } from '@/lib/accweb/types'
import { liveInstance } from '@/lib/accweb/client'
import { computed, onMounted, onUnmounted, ref, watch } from 'vue'
import InstanceStartStopBtn from '@/components/InstanceStartStopBtn.vue'
import moment from 'moment'

const { instanceId } = defineProps<{
  instanceId: string
}>()

const emit = defineEmits(['instanceUpdated'])

let intervalId: NodeJS.Timeout

const live = ref<LiveServerInstancePayload | null>(null)

function loadLiveInstance(id: string) {
  liveInstance(id)
    .then((resp) => {
      live.value = resp
    })
    .catch((err) => console.error('Error fetching live instance:', err))
}

const liveTs = computed(() => {
  return moment(live.value?.live?.updatedAt).format('MM/DD/YYYY, HH:mm:ss A')
})

const onInit = () => {
  clearInterval(intervalId)
  loadLiveInstance(instanceId)
  intervalId = setInterval(() => loadLiveInstance(instanceId), 2000)
}

const refreshInstance = () => {
  loadLiveInstance(instanceId)
  emit('instanceUpdated')
}

watch(
  () => instanceId,
  () => {
    live.value = null
    onInit()
  },
)

onMounted(() => {
  onInit()
})

onUnmounted(() => {
  clearInterval(intervalId)
})
</script>

<template>
  <div v-if="live" class="flex flex-row gap-4">
    <div class="flex flex-auto flex-row gap-4 border rounded-lg p-3 text-sm">
      <div class="flex-auto"><strong>Status:</strong> {{ live?.serverState }}</div>

      <div class="flex-auto"><strong>Track:</strong> {{ live?.track }}</div>

      <div class="flex-auto">
        <strong>Phase:</strong> {{ live?.sessionType }} ({{ live?.sessionPhase }}) [{{
          live?.sessionRemaining
        }}
        min]
      </div>

      <div class="flex-auto"><strong>Drivers:</strong> {{ live?.nrClients }}</div>

      <div class="flex-auto"><strong>Last Update:</strong> {{ liveTs }}</div>
    </div>

    <div class="flex flex-none">
      <InstanceStartStopBtn
        :instanceId="live?.id!"
        :isRunning="live?.isRunning!"
        :refreshFunction="() => refreshInstance()"
      />
    </div>
  </div>

  <div v-else>Loading...</div>
</template>
