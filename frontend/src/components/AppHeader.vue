<script setup lang="ts">
import { computed } from 'vue'
import type { NavigationMenuItem } from '@nuxt/ui'
import { useRoute } from 'vue-router'
import Logo from '@/assets/logo.png'

const route = useRoute()

const isLoginPage = computed(() => route.path === '/login')

const items = computed<NavigationMenuItem[]>(() => [
  {
    label: 'Dedicated Servers',
    to: '/',
    active: route.path === '/',
  },
  {
    label: 'Settings',
    to: '/settings',
    active: route.path === '/settings',
  },
])
</script>

<template>
  <!-- <UHeader class="bg-red-800 mb-3"> -->
  <UHeader class="accweb-header" mode="slideover">
    <template #title>
      <img :src="Logo" alt="ACCweb" class="h-12" />
      <div id="title-description" class="mb-3 invisible md:visible">Server Management</div>
    </template>

    <UNavigationMenu
      :items="items"
      v-if="!isLoginPage"
      :ui="{ link: 'data-[active]:text-red-400' }"
    />

    <template #right>
      <UTooltip text="Open on GitHub">
        <UButton
          color="neutral"
          variant="ghost"
          to="https://github.com/assetto-corsa-web/accweb"
          target="_blank"
          icon="i-simple-icons:github"
          aria-label="GitHub"
          class="invisible lg:visible"
        />
      </UTooltip>

      <UTooltip text="Discord">
        <UButton
          color="neutral"
          variant="ghost"
          to="https://discord.gg/AVWdF56t6c"
          target="_blank"
          icon="i-simple-icons:discord"
          aria-label="Discord"
          class="invisible lg:visible"
        />
      </UTooltip>

      <USeparator
        orientation="vertical"
        class="h-10 mx-3 invisible lg:visible"
        v-if="!isLoginPage"
      />

      <UUser name="John Doe" description="Administrator" v-if="!isLoginPage" />
    </template>

    <template #body>
      <UNavigationMenu :items="items" orientation="vertical" class="-mx-2.5" />
    </template>
  </UHeader>
</template>

<style scoped>
[data-slot='header'] #title-description {
  display: none;
}
</style>
