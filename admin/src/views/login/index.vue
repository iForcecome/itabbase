<template>
  <div class="login-screen">
    <!-- Left: Brand Panel -->
    <div class="brand-panel">
      <div class="brand-bg">
        <div class="grid-lines" />
        <div class="glow glow-1" />
        <div class="glow glow-2" />
        <div class="glow glow-3" />
        <div class="glow glow-4" />
      </div>

      <!-- Top-left brand logo -->
      <div class="panel-brand">
        <span class="panel-brand-icon">⬡</span>
        <span class="panel-brand-name">ITabBase</span>
      </div>

      <div class="brand-body">
        <!-- Slogan -->
        <div class="brand-top">
          <h1 class="slogan">让数据管理<br /><span class="slogan-highlight">简单而强大</span></h1>
          <p class="brand-desc">
            灵活的集合管理 · 精细化权限控制 · 实时协同编辑 · 为团队打造专业级数据管理体验
          </p>
        </div>

        <!-- Full dashboard mockup -->
        <div class="dashboard">
          <!-- Mockup title bar with brand -->
          <div class="dash-titlebar">
            <div class="dash-title-left">
              <div class="dash-dots">
                <span class="dot dot-r" />
                <span class="dot dot-y" />
                <span class="dot dot-g" />
              </div>
              <span class="dash-brand">⬡ ITabBase</span>
            </div>
            <div class="dash-nav">
              <span class="nav-item active">Overview</span>
              <span class="nav-item">Collections</span>
              <span class="nav-item">Users</span>
              <span class="nav-item">Settings</span>
            </div>
          </div>

          <!-- Stat row -->
          <div class="dash-stats">
            <div v-for="s in stats" :key="s.label" class="dash-stat">
              <div class="stat-head">
                <span class="stat-icon" :style="{ background: s.color }">{{ s.icon }}</span>
                <span class="stat-trend up" v-if="s.trend">{{ s.trend }}</span>
              </div>
              <span class="dash-stat-num">{{ s.value }}</span>
              <span class="dash-stat-label">{{ s.label }}</span>
            </div>
          </div>

          <!-- Main area -->
          <div class="dash-main">
            <!-- Left: table -->
            <div class="dash-table-wrap">
              <div class="section-head">
                <span class="section-title">Collections</span>
                <span class="section-badge">{{ tableRows.length }} items</span>
              </div>
              <div class="dash-table">
                <div class="dash-table-head">
                  <span>Name</span>
                  <span>Type</span>
                  <span>Records</span>
                  <span>Owner</span>
                  <span>Updated</span>
                  <span>Status</span>
                </div>
                <div v-for="(row, i) in tableRows" :key="i" class="dash-table-row">
                  <span class="row-name">
                    <span class="row-icon">{{ row.icon }}</span>
                    {{ row.name }}
                  </span>
                  <span class="row-dim">{{ row.type }}</span>
                  <span class="row-dim">{{ row.records }}</span>
                  <span class="row-dim">{{ row.owner }}</span>
                  <span class="row-dim">{{ row.updated }}</span>
                  <span>
                    <span class="row-badge" :class="row.status">{{ row.statusText }}</span>
                  </span>
                </div>
              </div>
            </div>

            <!-- Right: sidebar -->
            <div class="dash-side">
              <!-- Activity chart -->
              <div class="side-card">
                <div class="section-head">
                  <span class="section-title">Weekly Activity</span>
                </div>
                <div class="chart-labels">
                  <span v-for="d in ['Mon','Tue','Wed','Thu','Fri','Sat','Sun']" :key="d">{{ d }}</span>
                </div>
                <div class="chart-bars">
                  <div v-for="(h, i) in [45, 72, 56, 88, 64, 38, 80]" :key="i" class="chart-col">
                    <div class="chart-bar" :style="{ height: h + '%' }" />
                  </div>
                </div>
              </div>

              <!-- Roles -->
              <div class="side-card">
                <div class="section-head">
                  <span class="section-title">Team Roles</span>
                  <span class="section-badge">{{ roles.reduce((a, r) => a + r.count, 0) }} members</span>
                </div>
                <div class="role-list">
                  <div v-for="r in roles" :key="r.name" class="role-row">
                    <span class="role-dot" :style="{ background: r.color }" />
                    <span class="role-name">{{ r.name }}</span>
                    <div class="role-bar"><div :style="{ width: r.pct, background: r.color }" /></div>
                    <span class="role-count">{{ r.count }}</span>
                  </div>
                </div>
              </div>

              <!-- Recent actions -->
              <div class="side-card">
                <div class="section-head">
                  <span class="section-title">Recent Actions</span>
                </div>
                <div class="action-list">
                  <div v-for="a in actions" :key="a.text" class="action-row">
                    <span class="action-dot" :style="{ background: a.color }" />
                    <span class="action-text">{{ a.text }}</span>
                    <span class="action-time">{{ a.time }}</span>
                  </div>
                </div>
              </div>
            </div>
          </div>
        </div>

        <!-- Bottom feature tags -->
        <div class="feature-tags">
          <span class="ftag" v-for="f in features" :key="f">{{ f }}</span>
        </div>
      </div>
    </div>

    <!-- Right: Login Form -->
    <div class="form-panel">
      <div class="form-wrapper">
        <div style="margin-bottom: 32px">
          <h2 class="form-title">欢迎回来</h2>
          <a-typography-text type="secondary">登录以继续使用管理后台</a-typography-text>
        </div>

        <a-tabs v-model:activeKey="loginMode" centered>
          <a-tab-pane key="wps" tab="WPS 登录">
            <div class="tab-body">
              <a-button type="primary" block size="large" @click="loginWps">
                <LoginOutlined /> 使用 WPS 登录
              </a-button>
              <a-alert
                v-if="error"
                type="error"
                :message="error"
                show-icon
                closable
                style="margin-top: 16px"
                @close="error = ''"
              />
            </div>
          </a-tab-pane>

          <a-tab-pane key="local" tab="账号登录">
            <div class="tab-body">
              <a-form :model="loginForm" layout="vertical" @finish="loginLocal">
                <a-form-item name="username" label="用户名">
                  <a-input
                    v-model:value="loginForm.username"
                    placeholder="请输入用户名"
                    :disabled="submitting"
                    autocomplete="username"
                    size="large"
                  >
                    <template #prefix>
                      <UserOutlined style="color: rgba(0, 0, 0, 0.25)" />
                    </template>
                  </a-input>
                </a-form-item>
                <a-form-item name="password" label="密码">
                  <a-input-password
                    v-model:value="loginForm.password"
                    placeholder="请输入密码"
                    :disabled="submitting"
                    autocomplete="current-password"
                    size="large"
                  >
                    <template #prefix>
                      <LockOutlined style="color: rgba(0, 0, 0, 0.25)" />
                    </template>
                  </a-input-password>
                </a-form-item>
                <a-form-item style="margin-bottom: 0">
                  <a-button
                    type="primary"
                    html-type="submit"
                    block
                    size="large"
                    :loading="submitting"
                    :disabled="!loginForm.username || !loginForm.password"
                  >
                    登录
                  </a-button>
                </a-form-item>
                <a-alert
                  v-if="localError"
                  type="error"
                  :message="localError"
                  show-icon
                  closable
                  style="margin-top: 16px"
                  @close="localError = ''"
                />
              </a-form>
            </div>
          </a-tab-pane>
        </a-tabs>

        <div class="form-footer">
          <a-typography-text type="secondary" style="font-size: 12px">
            © {{ new Date().getFullYear() }} ITabBase
          </a-typography-text>
        </div>
      </div>
    </div>
  </div>
</template>

<script setup lang="ts">
import { api, loginURL } from "@/api";
import { useUserStore } from "@/stores/user";

const route = useRoute();
const router = useRouter();

const loginMode = ref<"wps" | "local">("wps");
const loginForm = reactive({ username: "", password: "" });
const localError = ref("");
const error = ref("");
const submitting = ref(false);

const stats = [
  { value: "1,284", label: "Total Records", icon: "📊", color: "rgba(96,165,250,0.2)", trend: "+12.5%" },
  { value: "36", label: "Active Users", icon: "👥", color: "rgba(52,211,153,0.2)", trend: "+3" },
  { value: "8", label: "Collections", icon: "📁", color: "rgba(251,191,36,0.2)", trend: "+2" },
  { value: "99.9%", label: "Uptime", icon: "⚡", color: "rgba(167,139,250,0.2)", trend: "" },
];

const tableRows = [
  { name: "users", icon: "👤", type: "auth", records: "36", owner: "admin", updated: "2 min ago", status: "active", statusText: "Active" },
  { name: "projects", icon: "📋", type: "data", records: "128", owner: "admin", updated: "5 min ago", status: "active", statusText: "Active" },
  { name: "permissions", icon: "🔐", type: "acl", records: "24", owner: "system", updated: "1 hr ago", status: "active", statusText: "Active" },
  { name: "configs", icon: "⚙️", type: "system", records: "12", owner: "system", updated: "3 hr ago", status: "active", statusText: "Active" },
  { name: "audit_logs", icon: "📝", type: "logs", records: "1,084", owner: "system", updated: "Just now", status: "active", statusText: "Active" },
  { name: "workflows", icon: "🔄", type: "data", records: "47", owner: "admin", updated: "30 min ago", status: "active", statusText: "Active" },
  { name: "templates", icon: "📄", type: "data", records: "15", owner: "editor", updated: "2 hr ago", status: "paused", statusText: "Paused" },
];

const roles = [
  { name: "Admin", pct: "100%", count: 4, color: "rgba(96,165,250,0.7)" },
  { name: "Editor", pct: "65%", count: 14, color: "rgba(52,211,153,0.7)" },
  { name: "Viewer", pct: "45%", count: 18, color: "rgba(167,139,250,0.7)" },
];

const actions = [
  { text: "admin updated projects", time: "2m", color: "#60a5fa" },
  { text: "editor created new record", time: "5m", color: "#34d399" },
  { text: "system backup completed", time: "1h", color: "#a78bfa" },
  { text: "admin modified ACL rules", time: "3h", color: "#fbbf24" },
];

const features = [
  "🗂 集合管理", "🔐 权限控制", "👥 团队协作", "📊 数据分析", "⚡ 实时同步", "🔄 工作流",
];

function loginWps() {
  window.location.href = loginURL(window.location.href);
}

async function loginLocal() {
  localError.value = "";
  submitting.value = true;
  try {
    await api.localLogin(loginForm.username, loginForm.password);
    loginForm.password = "";
    const userStore = useUserStore();
    await userStore.init();
    if (!userStore.isAdmin) {
      localError.value = "管理后台仅限管理员登录";
      await userStore.logout();
      return;
    }
    const redirect = (route.query.redirect as string) || "/";
    router.replace(redirect);
  } catch (err) {
    localError.value = err instanceof Error ? err.message : String(err);
  } finally {
    submitting.value = false;
  }
}
</script>

<style scoped>
.login-screen {
  display: flex;
  min-height: 100vh;
}

/* ══════════════════════════════════
   Left Brand Panel
   ══════════════════════════════════ */
.brand-panel {
  flex: 1.2;
  position: relative;
  overflow: hidden;
  background: linear-gradient(145deg, #080c16 0%, #111b30 40%, #0d1526 100%);
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 36px 40px;
}

.brand-bg {
  position: absolute;
  inset: 0;
  pointer-events: none;
}
.grid-lines {
  position: absolute;
  inset: 0;
  background-image:
    linear-gradient(rgba(255, 255, 255, 0.02) 1px, transparent 1px),
    linear-gradient(90deg, rgba(255, 255, 255, 0.02) 1px, transparent 1px);
  background-size: 40px 40px;
  mask-image: radial-gradient(ellipse 90% 80% at 50% 50%, black 20%, transparent 100%);
  -webkit-mask-image: radial-gradient(ellipse 90% 80% at 50% 50%, black 20%, transparent 100%);
}
.glow {
  position: absolute;
  border-radius: 50%;
  filter: blur(120px);
  animation: drift 12s ease-in-out infinite;
}
.glow-1 {
  width: 600px; height: 600px;
  background: rgba(59, 130, 246, 0.1);
  top: -20%; right: -15%;
}
.glow-2 {
  width: 500px; height: 500px;
  background: rgba(139, 92, 246, 0.08);
  bottom: -15%; left: -10%;
  animation-delay: -4s;
}
.glow-3 {
  width: 400px; height: 400px;
  background: rgba(6, 182, 212, 0.06);
  top: 30%; left: 30%;
  animation-delay: -8s;
}
.glow-4 {
  width: 350px; height: 350px;
  background: rgba(251, 191, 36, 0.04);
  bottom: 20%; right: 20%;
  animation-delay: -6s;
}
@keyframes drift {
  0%, 100% { transform: translate(0, 0) scale(1); }
  33% { transform: translate(25px, -20px) scale(1.06); }
  66% { transform: translate(-18px, 12px) scale(0.96); }
}

/* ── Top-left brand ── */
.panel-brand {
  position: absolute;
  top: 28px;
  left: 36px;
  z-index: 10;
  display: flex;
  align-items: center;
  gap: 12px;
}
.panel-brand-icon {
  font-size: 32px;
  color: #60a5fa;
  filter: drop-shadow(0 0 14px rgba(96, 165, 250, 0.6));
}
.panel-brand-name {
  font-size: 24px;
  font-weight: 800;
  color: rgba(255, 255, 255, 0.85);
  letter-spacing: -0.02em;
  text-shadow: 0 0 20px rgba(96, 165, 250, 0.15);
}

/* ── Brand body ── */
.brand-body {
  position: relative;
  z-index: 2;
  width: 100%;
  max-width: 820px;
  display: flex;
  flex-direction: column;
  gap: 28px;
}

.brand-top { text-align: center; }
.slogan {
  margin: 0;
  font-size: 42px;
  font-weight: 800;
  line-height: 1.25;
  color: #fff;
  letter-spacing: -0.03em;
}
.slogan-highlight {
  background: linear-gradient(135deg, #60a5fa, #a78bfa, #34d399);
  -webkit-background-clip: text;
  -webkit-text-fill-color: transparent;
  background-clip: text;
}
.brand-desc {
  margin: 14px auto 0;
  max-width: 480px;
  font-size: 14px;
  line-height: 1.8;
  color: rgba(148, 163, 184, 0.5);
  letter-spacing: 0.03em;
}

/* ══════════════════════════════════
   Dashboard mockup
   ══════════════════════════════════ */
.dashboard {
  background: rgba(15, 23, 42, 0.55);
  backdrop-filter: blur(24px);
  border: 1px solid rgba(255, 255, 255, 0.06);
  border-radius: 14px;
  overflow: hidden;
  box-shadow:
    0 30px 100px rgba(0, 0, 0, 0.5),
    0 0 0 1px rgba(255, 255, 255, 0.04),
    inset 0 1px 0 rgba(255, 255, 255, 0.06);
  display: flex;
  flex-direction: column;
  animation: mockup-in 0.8s ease-out both;
}
@keyframes mockup-in {
  from { opacity: 0; transform: translateY(24px) scale(0.96); }
  to { opacity: 1; transform: translateY(0) scale(1); }
}

/* ── Title bar ── */
.dash-titlebar {
  display: flex;
  align-items: center;
  justify-content: space-between;
  padding: 10px 18px;
  background: rgba(255, 255, 255, 0.03);
  border-bottom: 1px solid rgba(255, 255, 255, 0.05);
}
.dash-title-left {
  display: flex;
  align-items: center;
  gap: 12px;
}
.dash-dots {
  display: flex;
  gap: 6px;
}
.dot {
  width: 10px; height: 10px;
  border-radius: 50%;
  background: rgba(255,255,255,0.1);
}
.dot-r { background: #ef4444; }
.dot-y { background: #eab308; }
.dot-g { background: #22c55e; }
.dash-brand {
  font-size: 12px;
  font-weight: 700;
  color: rgba(224, 231, 240, 0.75);
  letter-spacing: -0.01em;
}
.dash-nav {
  display: flex;
  gap: 20px;
}
.nav-item {
  font-size: 11px;
  color: rgba(148, 163, 184, 0.45);
  cursor: default;
  padding: 2px 0;
  letter-spacing: 0.02em;
}
.nav-item.active {
  color: rgba(224, 231, 240, 0.85);
  border-bottom: 1.5px solid rgba(96, 165, 250, 0.6);
}

/* ── Stats ── */
.dash-stats {
  display: grid;
  grid-template-columns: repeat(4, 1fr);
  gap: 12px;
  padding: 16px 18px;
}
.dash-stat {
  background: rgba(255, 255, 255, 0.025);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 10px;
  padding: 12px 14px;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.stat-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
}
.stat-icon {
  width: 26px; height: 26px;
  border-radius: 7px;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 13px;
}
.stat-trend {
  font-size: 10px;
  font-weight: 600;
  color: #34d399;
}
.dash-stat-num {
  font-size: 22px;
  font-weight: 700;
  color: #f0f4ff;
  font-variant-numeric: tabular-nums;
  letter-spacing: -0.02em;
}
.dash-stat-label {
  font-size: 10px;
  color: rgba(148, 163, 184, 0.45);
  text-transform: uppercase;
  letter-spacing: 0.05em;
}

/* ── Section head ── */
.section-head {
  display: flex;
  align-items: center;
  justify-content: space-between;
  margin-bottom: 10px;
}
.section-title {
  font-size: 11px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: rgba(203, 213, 225, 0.6);
}
.section-badge {
  font-size: 10px;
  padding: 2px 8px;
  border-radius: 8px;
  background: rgba(255,255,255,0.04);
  color: rgba(148, 163, 184, 0.5);
}

/* ── Main grid ── */
.dash-main {
  display: grid;
  grid-template-columns: 1fr 230px;
  gap: 14px;
  padding: 2px 18px 18px;
}

/* ── Table ── */
.dash-table-wrap { min-width: 0; }
.dash-table {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 10px;
  overflow: hidden;
}
.dash-table-head {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1.2fr 0.8fr;
  padding: 9px 14px;
  background: rgba(255, 255, 255, 0.025);
  border-bottom: 1px solid rgba(255, 255, 255, 0.04);
  font-size: 10px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.06em;
  color: rgba(148, 163, 184, 0.4);
}
.dash-table-row {
  display: grid;
  grid-template-columns: 2fr 1fr 1fr 1fr 1.2fr 0.8fr;
  padding: 8px 14px;
  border-bottom: 1px solid rgba(255, 255, 255, 0.025);
  font-size: 11px;
  color: rgba(203, 213, 225, 0.7);
  align-items: center;
}
.dash-table-row:last-child { border-bottom: none; }
.dash-table-row:hover { background: rgba(255,255,255,0.015); }
.row-name {
  color: #e2e8f0;
  font-weight: 500;
  display: flex;
  align-items: center;
  gap: 6px;
}
.row-icon { font-size: 13px; }
.row-dim { color: rgba(148, 163, 184, 0.45); }
.row-badge {
  display: inline-block;
  padding: 2px 8px;
  border-radius: 9px;
  font-size: 9px;
  font-weight: 600;
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.row-badge.active {
  background: rgba(52, 211, 153, 0.12);
  color: #34d399;
}
.row-badge.paused {
  background: rgba(251, 191, 36, 0.12);
  color: #fbbf24;
}

/* ── Sidebar ── */
.dash-side {
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.side-card {
  background: rgba(255, 255, 255, 0.02);
  border: 1px solid rgba(255, 255, 255, 0.04);
  border-radius: 10px;
  padding: 12px;
  display: flex;
  flex-direction: column;
}

/* ── Chart ── */
.chart-labels {
  display: flex;
  justify-content: space-between;
  margin-bottom: 6px;
}
.chart-labels span {
  font-size: 8px;
  color: rgba(148,163,184,0.35);
  text-transform: uppercase;
  letter-spacing: 0.04em;
}
.chart-bars {
  display: flex;
  align-items: flex-end;
  gap: 5px;
  height: 56px;
}
.chart-col {
  flex: 1;
  height: 100%;
  display: flex;
  align-items: flex-end;
}
.chart-bar {
  width: 100%;
  border-radius: 3px 3px 0 0;
  background: linear-gradient(180deg, rgba(96, 165, 250, 0.6), rgba(96, 165, 250, 0.08));
  transition: height 0.5s ease;
}

/* ── Roles ── */
.role-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.role-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.role-dot {
  width: 7px; height: 7px;
  border-radius: 50%;
  flex-shrink: 0;
}
.role-name {
  font-size: 10px;
  color: rgba(203, 213, 225, 0.6);
  width: 38px;
  flex-shrink: 0;
}
.role-bar {
  flex: 1;
  height: 5px;
  border-radius: 3px;
  background: rgba(255, 255, 255, 0.04);
  overflow: hidden;
}
.role-bar div {
  height: 100%;
  border-radius: 3px;
}
.role-count {
  font-size: 10px;
  color: rgba(148, 163, 184, 0.4);
  width: 18px;
  text-align: right;
  font-variant-numeric: tabular-nums;
}

/* ── Actions ── */
.action-list {
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.action-row {
  display: flex;
  align-items: center;
  gap: 8px;
}
.action-dot {
  width: 6px; height: 6px;
  border-radius: 50%;
  flex-shrink: 0;
}
.action-text {
  font-size: 10px;
  color: rgba(203, 213, 225, 0.55);
  flex: 1;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.action-time {
  font-size: 9px;
  color: rgba(148, 163, 184, 0.35);
  flex-shrink: 0;
}

/* ── Feature tags ── */
.feature-tags {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 10px;
}
.ftag {
  padding: 5px 14px;
  border-radius: 20px;
  font-size: 11px;
  color: rgba(203, 213, 225, 0.5);
  background: rgba(255, 255, 255, 0.03);
  border: 1px solid rgba(255, 255, 255, 0.05);
  letter-spacing: 0.02em;
}

/* ══════════════════════════════════
   Right Form Panel
   ══════════════════════════════════ */
.form-panel {
  width: 440px;
  min-width: 440px;
  display: flex;
  align-items: flex-start;
  justify-content: center;
  padding: 60px 48px;
  padding-top: 18vh;
  background: #fff;
}
.form-wrapper {
  width: 100%;
  max-width: 360px;
}
.form-title {
  margin: 0 0 8px;
  font-size: 26px;
  font-weight: 700;
  color: rgba(0, 0, 0, 0.88);
  letter-spacing: -0.02em;
}
.tab-body {
  height: 250px;
  display: flex;
  flex-direction: column;
  justify-content: center;
}
.form-footer {
  text-align: center;
  margin-top: 40px;
}

/* ── Responsive ── */
@media (max-width: 1100px) {
  .slogan { font-size: 34px; }
  .brand-body { max-width: 640px; }
}
@media (max-width: 960px) {
  .login-screen { flex-direction: column; }
  .brand-panel {
    flex: none;
    min-height: 420px;
    padding: 24px;
  }
  .slogan { font-size: 28px; }
  .brand-desc { display: none; }
  .dash-stats { grid-template-columns: repeat(2, 1fr); }
  .dash-main { grid-template-columns: 1fr; }
  .dash-side { display: none; }
  .feature-tags { display: none; }
  .form-panel {
    width: 100%;
    min-width: unset;
    flex: 1;
    padding: 40px;
    padding-top: 40px;
    align-items: center;
  }
}
</style>
