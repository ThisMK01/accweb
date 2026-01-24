<script setup lang="ts">
import { ref, onMounted, onUnmounted, computed } from 'vue'
import { fetchServer } from '@/lib/accweb/client'
import type { ListServerItem } from '@/lib/accweb/types'
import InstanceCard from './home-page/InstanceCard.vue'
import _ from 'lodash'

const servers = ref<ListServerItem[]>([])
let intervalId: NodeJS.Timeout

onMounted(async () => {
  refreshServers()

  // intervalId = setInterval(refreshServers, 10000)
})

onUnmounted(() => {
  clearInterval(intervalId)
})

const refreshServers = () => {
  fetchServer()
    .then((serversResp) => {
      servers.value = serversResp
      intervalId = setTimeout(refreshServers, 10000)
    })
    .catch((error) => {
      console.error('Error fetching servers:', error)
    })
}

const sortedServerList = computed<ListServerItem[]>(() => {
  return _.sortBy(servers.value, ['name'])
})
</script>

<template>
  <div class="flex gap-3">
    <div class="flex-auto">
      <div v-if="servers.length">
        <div class="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
          <InstanceCard
            v-for="server in sortedServerList"
            :key="server.id + '-' + server.isRunning"
            :instance="server"
            :refreshFunction="refreshServers"
          />
        </div>
      </div>
      <div v-else>
        <p>No servers found.</p>
      </div>
    </div>
  </div>
</template>
