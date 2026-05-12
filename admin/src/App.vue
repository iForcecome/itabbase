<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter, RouterView, RouterLink } from 'vue-router'
import { api, loginURL, UnauthenticatedError, type MetaCollection, type User } from './api'

const route = useRoute()
const router = useRouter()

const loading = ref(true)
const user = ref<User | null>(null)
const collections = ref<MetaCollection[]>([])
const error = ref('')

const pendingApplication = ref(false)
const loginMode = ref<'wps' | 'local'>('wps')
const localUsername = ref('')
const localPassword = ref('')
const localError = ref('')
const localSubmitting = ref(false)

async function load() {
  loading.value = true
  error.value = ''
  // SSO callback redirects pending users here with ?pending=1.
  // Render the waiting view without hitting whoami (no cookie set).
  const params = new URLSearchParams(window.location.search)
  if (params.get('pending') === '1') {
    pendingApplication.value = true
    user.value = null
    loading.value = false
    return
  }
  pendingApplication.value = false
  try {
    const me = await api.whoami()
    user.value = me.data
    const cols = await api.collections()
    collections.value = cols.data
    if (route.path === '/welcome' && collections.value.length > 0) {
      router.replace(`/c/${collections.value[0].name}`)
    }
  } catch (err) {
    if (err instanceof UnauthenticatedError) {
      user.value = null
    } else {
      error.value = err instanceof Error ? err.message : String(err)
    }
  } finally {
    loading.value = false
  }
}

function dismissPending() {
  // Strip ?pending=1 and reload back to a clean login screen.
  const url = new URL(window.location.href)
  url.searchParams.delete('pending')
  window.location.href = url.toString()
}

async function logout() {
  await api.logout()
  user.value = null
  collections.value = []
  router.replace('/welcome')
}

function login() {
  window.location.href = loginURL(window.location.href)
}

async function loginLocal() {
  localError.value = ''
  localSubmitting.value = true
  try {
    await api.localLogin(localUsername.value, localPassword.value)
    localPassword.value = ''
    await load()
  } catch (err) {
    localError.value = err instanceof Error ? err.message : String(err)
  } finally {
    localSubmitting.value = false
  }
}

onMounted(load)
</script>

<template>
  <div class="app">
    <header class="topbar">
      <span class="brand">itab admin</span>
      <span class="spacer" />
      <template v-if="user">
        <span class="user">{{ user.name }}</span>
        <button class="link" @click="logout">登出</button>
      </template>
    </header>

    <div v-if="loading" class="loading">加载中…</div>

    <div v-else-if="pendingApplication" class="login-prompt">
      <h2>申请已提交</h2>
      <p class="subtitle">你的访问申请正在等待管理员审批。</p>
      <p class="hint">审批通过后再次登录即可进入系统。如需加急,请联系管理员。</p>
      <button class="btn btn-block" @click="dismissPending">返回登录</button>
    </div>

    <div v-else-if="!user" class="login-prompt">
      <template v-if="loginMode === 'wps'">
        <h2>登录</h2>
        <p class="subtitle">使用 WPS 账号登录管理后台。</p>

        <button class="btn primary btn-block" @click="login">使用 WPS 登录</button>

        <button class="btn-link" @click="loginMode = 'local'">其他登录方式 →</button>

        <p v-if="error" class="form-error">{{ error }}</p>
      </template>

      <template v-else>
        <button class="back-link" @click="loginMode = 'wps'">← 返回</button>
        <h2>本地账号登录</h2>
        <p class="subtitle">使用管理员账号密码登录。</p>

        <form class="local-form" @submit.prevent="loginLocal" autocomplete="on">
          <label class="field">
            <span>用户名</span>
            <input v-model="localUsername" autocomplete="username" :disabled="localSubmitting" />
          </label>
          <label class="field">
            <span>密码</span>
            <input
              v-model="localPassword"
              type="password"
              autocomplete="current-password"
              :disabled="localSubmitting"
            />
          </label>
          <button
            type="submit"
            class="btn primary btn-block"
            :disabled="localSubmitting || !localUsername || !localPassword"
          >
            {{ localSubmitting ? '登录中…' : '登录' }}
          </button>
          <p v-if="localError" class="form-error">{{ localError }}</p>
        </form>
      </template>
    </div>

    <div v-else class="layout">
      <aside class="sidebar">
        <h3>集合</h3>
        <ul>
          <li v-for="c in collections" :key="c.name">
            <RouterLink :to="`/c/${c.name}`" class="nav-link">
              <span class="display">{{ c.display || c.name }}</span>
              <span class="name">{{ c.name }}</span>
            </RouterLink>
          </li>
        </ul>
        <p v-if="collections.length === 0" class="empty-mini">无 collection</p>
      </aside>
      <main class="main">
        <RouterView />
      </main>
    </div>
  </div>
</template>
