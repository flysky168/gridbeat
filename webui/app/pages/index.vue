<script setup lang="ts">
definePageMeta({ layout: 'dashboard' })


const { signOut } = useAuth()

const loggingOut = ref(false)

async function onLogout() {
  if (loggingOut.value) return
  loggingOut.value = true
  try {
    // 退出并回到登录页（你也可以改成 '/' 或其他页面）
    await signOut({ callbackUrl: '/login' })
  } finally {
    loggingOut.value = false
  }
}
</script>

<template>
  <UDashboardNavbar title="GridBeat Gateway">
    <template #right>
      <div class="flex items-center gap-2">
        <!-- 语言切换（你已有就保留） -->
        <!--
        <ULocaleSelect
          :model-value="locale"
          @update:model-value="setLocale"
          size="sm"
          class="w-32"
        />
        -->

        <!-- 退出登录 -->
        <UButton
          size="sm"
          color="neutral"
          variant="ghost"
          icon="i-lucide-log-out"
          label="退出登录"
          :loading="loggingOut"
          :disabled="loggingOut"
          @click="onLogout"
        />
      </div>
    </template>
  </UDashboardNavbar>
</template>