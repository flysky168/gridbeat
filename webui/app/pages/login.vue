<script setup lang="ts">
definePageMeta({
  // 用 nuxt-auth 全站保护时，登录页一定要放行且只允许未登录访问
  auth: {
    unauthenticatedOnly: true,
    navigateAuthenticatedTo: '/',
  },
  layout: 'login', // 推荐给登录页一个空布局（如果你有）
})

const { t, locale,setLocale } = useI18n()
const route = useRoute()
const redirect = computed(() => (route.query.redirect as string) || '/')

const { signIn, status } = useAuth()

const form = reactive({
  username: '',
  password: '',
})

const errorMsg = ref<string>('')

const allowed = new Set(['en', 'zh', 'ja'] as const)
type AppLocale = 'en' | 'zh' | 'ja'

function onLocaleChange(v: string) {
  if (allowed.has(v as AppLocale)) setLocale(v as AppLocale)
}
const loading = computed(() => status.value === 'loading')

async function onSubmit() {
  errorMsg.value = ''
  try {

   // local provider: credentials signIn
    await signIn(
      { username: form.username, password: form.password },
      { callbackUrl: redirect.value }
    )
  } catch (e: any) {
    // nuxt-auth 抛错形式可能不同，这里做容错显示
    errorMsg.value = e?.data?.statusMessage || e?.message || 'Login failed'
  }
}
</script>

<template>
  <div class="min-h-screen flex items-center justify-center px-4">
    <UCard class="w-full max-w-md">
      <template #header>
        <div class="flex items-center justify-between">
          <div class="text-lg font-semibold">
            {{ t('login.title') || 'Sign in' }}
          </div>

          <!-- 语言选择（可选） -->

  <USelect
    :model-value="locale"
    :items="[
      { label: 'English', value: 'en' },
      { label: '中文', value: 'zh' },
      { label: '日本語', value: 'ja' },
    ]"
    size="sm"
    class="w-32"
    @update:model-value="onLocaleChange"
  />

        </div>
      </template>

      <UForm :state="form" class="space-y-4" @submit="onSubmit">

  <UFormField :label="t('login.username', 'Username')" name="username" required>
    <UInput v-model="form.username" autocomplete="username" />
  </UFormField>

  <UFormField :label="t('login.password', 'Password')" name="password" required>
    <UInput v-model="form.password" type="password" autocomplete="current-password" />
  </UFormField>

<UAlert
  v-if="errorMsg"
  icon="i-heroicons-exclamation-triangle"
  color="error"
  variant="soft"
  :title="errorMsg"
/>

        <UButton type="submit" block :loading="loading">
          {{ t('login.submit') || 'Login' }}
        </UButton>
      </UForm>

      <template #footer>
        <div class="text-xs text-gray-500">
          {{ t('app.title') || 'Industrial Gateway Admin' }}
        </div>
      </template>
    </UCard>
  </div>
</template>
