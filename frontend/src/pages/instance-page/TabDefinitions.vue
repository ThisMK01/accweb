<script setup lang="ts">
import { saveInstance } from '@/lib/accweb/client'
import type { InstancePayload } from '@/lib/accweb/types'
import type { FormError, TabsItem } from '@nuxt/ui'
import { ref, watch, computed } from 'vue'
import { schema } from '@/lib/accweb/schema'
import DefAccweb from './definitions/DefAccweb.vue'
import DefSettings from './definitions/DefSettings.vue'
import DefGeneral from './definitions/DefGeneral.vue'
import _ from 'lodash'
import DefEvent from './definitions/DefEvent.vue'

const toast = useToast()

const props = defineProps<{ instance: InstancePayload }>()
const data = ref<InstancePayload>(_.cloneDeep(props.instance))
const snapshot = ref<InstancePayload>(_.cloneDeep(props.instance))

const emit = defineEmits(['instanceUpdated'])

const isDirty = computed(() => !_.isEqual(data.value, snapshot.value))

watch(
  () => props.instance,
  (newValue) => {
    if (newValue.id != data.value.id) {
      data.value = _.cloneDeep(newValue)
      snapshot.value = _.cloneDeep(newValue)
    }
  },
)

const tabItems = ref<TabsItem[]>([
  {
    label: 'ACCWeb',
    content: 'This is the accweb content.',
    slot: 'accweb',
  },
  {
    label: 'General',
    content: 'This is the general content.',
    slot: 'general',
  },
  {
    label: 'Settings',
    content: 'This is the general content.',
    slot: 'settings',
  },
  {
    label: 'Event',
    content: 'This is the statistics content.',
    slot: 'event',
  },
  {
    label: 'Rules',
    content: 'This is the general content.',
    slot: 'rules',
  },
  {
    label: 'Entry List',
    content: 'This is the general content.',
    slot: 'entrylist',
  },
  {
    label: 'BoP',
    content: 'This is the bop content.',
    slot: 'bop',
  },
  {
    label: 'Assists',
    content: 'This is the general content.',
    slot: 'assists',
  },
])

function validate(): FormError[] {
  const errors: FormError[] = []

  const result = schema.safeParse(data.value)

  if (result.success) {
    return []
  }

  _.forEach(result.error.issues, (err) => {
    errors.push({ name: err.path.join('.'), message: err.message })
  })

  console.log('errors', errors)

  return errors
}

async function onSubmit() {
  saveInstance(data.value)
    .then(() => {
      snapshot.value = _.cloneDeep(data.value)
      toast.add({ title: 'Success', description: 'The form has been submitted.', color: 'success' })
      emit('instanceUpdated')
    })
    .catch((err) => {
      toast.add({
        title: 'Failed to save',
        description: err.response.data.error,
        color: 'error',
      })
    })
}
</script>

<template>
  <div v-if="data">
    <UForm :validate="validate" :state="data" class="space-y-4" @submit="onSubmit">
      <div class="mb-4 flex flex-row gap-4">
        <div class="flex-auto">
          {{ isDirty }}
          <UFormField label="Server name" name="acc.settings.serverName">
            <UInput v-model="data.acc.settings.serverName" class="w-full" />
          </UFormField>
        </div>

        <div class="flex-none self-end">
          <UButton type="submit" :disabled="instance.is_running || !isDirty">Save Changes</UButton>
        </div>
      </div>

      <UTabs
        :items="tabItems"
        class="w-full gap-4"
        variant="link"
        :unmount-on-hide="false"
        :ui="{}"
      >
        <template #accweb>
          <DefAccweb v-model="data.accWeb" />
        </template>

        <template #general>
          <DefGeneral v-model="data.acc.configuration" />
        </template>

        <template #settings>
          <DefSettings v-model="data.acc.settings" />
        </template>

        <template #event>
          <DefEvent v-model="data.acc.event" />
        </template>

        <template #rules>
          <div class="p-4">Rules Settings Content</div>
        </template>

        <template #entrylist>
          <div class="p-4">Entry List Settings Content</div>
        </template>

        <template #bop>
          <div class="p-4">BoP Settings Content</div>
        </template>

        <template #assists>
          <div class="p-4">Assists Settings Content</div>
        </template>
      </UTabs>
    </UForm>
  </div>

  <div v-else>Loading...</div>
</template>
