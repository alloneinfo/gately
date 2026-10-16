<template>
  <div class="layout" :class="{ collapsed }">
    <aside v-if="!isMobile" class="aside">
      <SideNav :collapsed="collapsed" />
      <button class="collapse-btn" :title="collapsed ? '展开侧栏' : '收起侧栏'" @click="toggleCollapse">
        <PanelLeftOpen v-if="collapsed" :size="18" />
        <PanelLeftClose v-else :size="18" />
      </button>
    </aside>
    <el-drawer v-else v-model="drawer" direction="ltr" size="260px" :with-header="false" class="nav-drawer">
      <SideNav @navigate="drawer = false" />
    </el-drawer>

    <div class="main">
      <header class="header">
        <div class="header-left">
          <button v-if="isMobile" class="icon-btn" aria-label="打开菜单" @click="drawer = true">
            <Menu :size="20" />
          </button>
          <template v-if="isMobile">
            <AppLogo :size="26" />
            <span class="header-title">Gately</span>
          </template>
          <template v-else>
            <span v-if="group" class="crumb">{{ group }}</span>
            <ChevronRight v-if="group" :size="14" class="crumb-sep" />
            <span class="header-title">{{ route.meta.title }}</span>
          </template>
        </div>
        <div class="header-right">
          <el-tooltip :content="isDark ? '切换到亮色' : '切换到暗色'" :show-after="300">
            <button class="icon-btn" aria-label="切换主题" @click="toggle">
              <Sun v-if="isDark" :size="18" />
              <Moon v-else :size="18" />
            </button>
          </el-tooltip>
          <el-dropdown trigger="click" @command="onCommand">
            <button class="user-btn">
              <span class="avatar">{{ (username || 'A').slice(0, 1).toUpperCase() }}</span>
              <span v-if="!isMobile" class="user-name">{{ username }}</span>
              <ChevronDown :size="14" class="muted" />
            </button>
            <template #dropdown>
              <el-dropdown-menu>
                <el-dropdown-item command="password" :icon="KeyRound">修改密码</el-dropdown-item>
                <el-dropdown-item command="settings" :icon="Settings">系统设置</el-dropdown-item>
                <el-dropdown-item command="logout" :icon="LogOut" divided>退出登录</el-dropdown-item>
              </el-dropdown-menu>
            </template>
          </el-dropdown>
        </div>
      </header>
      <main class="content">
        <router-view />
      </main>
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref, watch } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { ChevronDown, ChevronRight, KeyRound, LogOut, Menu, Moon, PanelLeftClose, PanelLeftOpen, Settings, Sun } from 'lucide-vue-next'
import { api } from '../api'
import SideNav from '../components/SideNav.vue'
import AppLogo from '../components/AppLogo.vue'
import { groupByPath } from '../nav'
import { useIsMobile } from '../composables/useIsMobile'
import { useTheme } from '../composables/useTheme'

const route = useRoute()
const router = useRouter()
const isMobile = useIsMobile()
const { isDark, toggle } = useTheme()
const username = ref('')
const drawer = ref(false)
const collapsed = ref(localStorage.getItem('gately-collapsed') === '1')
const group = computed(() => groupByPath(route.path))

function toggleCollapse() {
  collapsed.value = !collapsed.value
  localStorage.setItem('gately-collapsed', collapsed.value ? '1' : '0')
}

// 切换页面时关闭移动端抽屉
watch(
  () => route.path,
  () => {
    drawer.value = false
  },
)

async function onCommand(cmd: string) {
  if (cmd === 'logout') {
    await api.post('/api/logout').catch(() => undefined)
    router.replace({ name: 'login' })
  } else if (cmd === 'password') {
    router.push({ path: '/settings', query: { tab: 'account' } })
  } else if (cmd === 'settings') {
    router.push('/settings')
  }
}

onMounted(async () => {
  const s = await api.get<{ username: string }>('/api/session')
  username.value = s.username
})
</script>

<style scoped>
.layout {
  --aside-w: 228px;
  display: flex;
  min-height: 100%;
}
.layout.collapsed {
  --aside-w: 68px;
}
.aside {
  position: fixed;
  inset: 0 auto 0 0;
  width: var(--aside-w);
  border-right: 1px solid var(--g-border);
  background: var(--g-sidebar-bg);
  transition: width 0.2s ease;
  z-index: 20;
  display: flex;
  flex-direction: column;
}
.aside > :first-child {
  flex: 1;
  min-height: 0;
}
.collapse-btn {
  height: 44px;
  border: none;
  border-top: 1px solid var(--g-border);
  background: transparent;
  color: var(--el-text-color-secondary);
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
}
.collapse-btn:hover {
  color: var(--el-color-primary);
}
.main {
  flex: 1;
  min-width: 0;
  margin-left: var(--aside-w);
  transition: margin-left 0.2s ease;
  display: flex;
  flex-direction: column;
}
.header {
  position: sticky;
  top: 0;
  z-index: 10;
  height: 56px;
  display: flex;
  align-items: center;
  justify-content: space-between;
  gap: 12px;
  padding: 0 20px;
  border-bottom: 1px solid var(--g-border);
  background: var(--g-header-bg);
  backdrop-filter: saturate(180%) blur(10px);
  -webkit-backdrop-filter: saturate(180%) blur(10px);
}
.header-left,
.header-right {
  display: flex;
  align-items: center;
  gap: 8px;
  min-width: 0;
}
.crumb {
  font-size: 14px;
  color: var(--el-text-color-secondary);
  white-space: nowrap;
}
.crumb-sep {
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
.header-title {
  font-weight: 600;
  font-size: 15px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.icon-btn {
  width: 36px;
  height: 36px;
  border-radius: 8px;
  border: none;
  background: transparent;
  color: var(--el-text-color-regular);
  display: inline-flex;
  align-items: center;
  justify-content: center;
  cursor: pointer;
}
.icon-btn:hover {
  background: var(--el-fill-color-light);
  color: var(--el-text-color-primary);
}
.user-btn {
  display: inline-flex;
  align-items: center;
  gap: 8px;
  height: 36px;
  padding: 0 8px 0 4px;
  border-radius: 18px;
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
  color: var(--el-text-color-primary);
  cursor: pointer;
  font-size: 13px;
}
.user-btn:hover {
  border-color: var(--el-border-color);
}
.avatar {
  width: 28px;
  height: 28px;
  border-radius: 50%;
  background: var(--el-color-primary);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-weight: 600;
  font-size: 13px;
}
.user-name {
  max-width: 120px;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.content {
  flex: 1;
  width: 100%;
  max-width: 1440px;
  margin: 0 auto;
  padding: 20px 24px 32px;
  min-width: 0;
}
@media (max-width: 767.98px) {
  .main {
    margin-left: 0;
  }
  .header {
    padding: 0 8px 0 4px;
    height: 52px;
  }
  .content {
    padding: 14px 12px 24px;
  }
}
</style>

<style>
.nav-drawer .el-drawer__body {
  padding: 0;
}
</style>
