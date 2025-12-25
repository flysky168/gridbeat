<!-- src/views/Login.vue -->
<template>
  <div class="login-page d-flex align-items-center justify-content-center">
    <div class="login-card card shadow-sm">
      <div class="card-body">
        <!-- 标题 -->
        <div class="d-flex align-items-center mb-3">
          <font-awesome-icon
            icon="fa-solid fa-plug"
            class="me-2 text-primary"
          />
          <div>
            <h5 class="mb-0">{{ t('login.title') }}</h5>
            <small class="text-muted">
              {{ t('login.subtitle') }}
            </small>
          </div>
        </div>

        <!-- 语言下拉（在用户名上方） -->
        <div class="mb-3">
          <label class="form-label d-block">
            <font-awesome-icon icon="fa-solid fa-globe" class="me-1" />
            {{ t('common.language') }}
          </label>
          <select
            v-model="selectedLocale"
            class="form-select form-select-sm"
          >
            <option
              v-for="lang in languages"
              :key="lang.code"
              :value="lang.code"
            >
              {{ lang.label }}
            </option>
          </select>
        </div>

        <!-- 登录表单 -->
        <form @submit.prevent="onSubmit">
          <div class="mb-3">
            <label class="form-label">
              <font-awesome-icon icon="fa-solid fa-user" class="me-1" />
              {{ t('login.username') }}
            </label>
            <div class="input-group">
              <span class="input-group-text">
                <font-awesome-icon icon="fa-solid fa-user" />
              </span>
              <input
                v-model="loginForm.username"
                type="text"
                class="form-control"
                autocomplete="username"
                :placeholder="t('login.usernamePlaceholder')"
              />
            </div>
          </div>

          <div class="mb-3">
            <label class="form-label">
              <font-awesome-icon icon="fa-solid fa-lock" class="me-1" />
              {{ t('login.password') }}
            </label>
            <div class="input-group">
              <span class="input-group-text">
                <font-awesome-icon icon="fa-solid fa-lock" />
              </span>
              <input
                v-model="loginForm.password"
                type="password"
                class="form-control"
                autocomplete="current-password"
                :placeholder="t('login.passwordPlaceholder')"
              />
            </div>
          </div>

          <div class="d-flex justify-content-between align-items-center mb-3">
            <div class="form-check">
              <input
                id="remember"
                v-model="rememberMe"
                class="form-check-input"
                type="checkbox"
              />
              <label class="form-check-label" for="remember">
                {{ t('login.rememberMe') }}
              </label>
            </div>
          </div>

          <button
            type="submit"
            class="btn btn-primary w-100 d-flex align-items-center justify-content-center"
            :disabled="loading"
          >
            <span
              v-if="loading"
              class="spinner-border spinner-border-sm me-2"
              role="status"
              aria-hidden="true"
            />
            <font-awesome-icon
              v-else
              icon="fa-solid fa-right-to-bracket"
              class="me-2"
            />
            <span>{{ t('login.submit') }}</span>
          </button>
        </form>
      </div>

      <!-- 底部版本信息 -->
      <div class="card-footer text-center text-muted small">
        {{ t('app.footer') }} · v{{ appVersion }}
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { reactive, ref, computed, onMounted } from 'vue'
import { useRouter, useRoute } from 'vue-router'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { useAuthStore } from '@/store/auth'
import { loginApi } from '@/api/auth'
import type { Locale } from '@/plugins/i18n'
import { APP_VERSION } from '@/config/app'

const router = useRouter()
const route = useRoute()
const auth = useAuthStore()
const { t, locale } = useI18n()

const appVersion = APP_VERSION

// 登录表单：用户名默认 root
const loginForm = reactive({
  username: 'root',
  password: '',
})

const rememberMe = ref(false)
const loading = ref(false)

// 语言配置
const languages = [
  { code: 'zh-CN' as Locale, label: '简体中文' },
  { code: 'en' as Locale, label: 'English' },
  { code: 'ja' as Locale, label: '日本語' },
]

// 下拉绑定的当前语言（getter/setter 直接联动 i18n locale）
const selectedLocale = computed<Locale>({
  get: () => locale.value as Locale,
  set: (val: Locale) => {
    if (locale.value === val) return
    locale.value = val
    localStorage.setItem('locale', val)
  },
})

// 初始化：从 localStorage 读取语言和用户名
onMounted(() => {
  const savedLocale = localStorage.getItem('locale') as Locale | null
  if (savedLocale && languages.some((l) => l.code === savedLocale)) {
    locale.value = savedLocale
  }

  const savedUser = localStorage.getItem('login_username')
  if (savedUser) {
    loginForm.username = savedUser
    rememberMe.value = true
  } else {
    // 没有保存过用户名时，默认 root
    loginForm.username = 'root'
  }
})

// 提交登录
const onSubmit = async () => {
  if (!loginForm.username || !loginForm.password) {
    ElMessage.warning(t('login.missingCredentials'))
    return
  }

  loading.value = true
  try {
    const res = await loginApi({
      username: loginForm.username,
      password: loginForm.password,
    })

    // 记住用户名
    if (rememberMe.value) {
      localStorage.setItem('login_username', loginForm.username)
    } else {
      localStorage.removeItem('login_username')
    }

    await auth.login({
      token: res.data.token,
      user: res.data,
    })

    ElMessage.success(t('login.success'))
    const redirect = (route.query.redirect as string) || '/'
    router.push(redirect)
  } catch (e) {
    console.error(e)
    ElMessage.error(t('login.failed'))
  } finally {
    loading.value = false
  }
}
</script>

<style scoped>
.login-page {
  min-height: 100vh;
  background: #f5f6f8;
  padding: 1rem;
}

.login-card {
  width: 100%;
  max-width: 420px;
}

.card-footer {
  background-color: #f9fafb;
}
</style>