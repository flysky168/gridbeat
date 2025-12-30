<script setup lang="ts">
import * as uiLocales from '@nuxt/ui/locale'

const { locale } = useI18n()

const uiLocale = computed(() => (uiLocales as any)[locale.value] || (uiLocales as any).en)
</script>

<template>
  <UApp :locale="uiLocale">
    <NuxtLayout>
      <NuxtPage />
    </NuxtLayout>
  </UApp>
</template>