<template>
  <el-container class="layout">
    <el-aside width="210px" class="aside">
      <div class="brand"><span class="logo">G</span>Gately</div>
      <el-menu :default-active="route.path" router class="menu">
        <el-menu-item index="/overview">
          <el-icon><Odometer /></el-icon><span>总览</span>
        </el-menu-item>
        <el-menu-item index="/portforward">
          <el-icon><Switch /></el-icon><span>端口转发</span>
        </el-menu-item>
        <el-menu-item v-for="item in planned" :key="item.title" :index="item.title" disabled>
          <el-icon><component :is="item.icon" /></el-icon>
          <span>{{ item.title }}</span>
          <el-tag size="small" type="info" class="soon">规划中</el-tag>
        </el-menu-item>
        <el-menu-item index="/settings">
          <el-icon><Setting /></el-icon><span>设置</span>
        </el-menu-item>
      </el-menu>
    </el-aside>
    <el-container>
      <el-header class="header">
        <span class="title">{{ route.meta.title }}</span>
        <div class="actions">
          <el-switch v-model="dark" inline-prompt active-text="暗" inactive-text="亮" @change="applyDark" />
          <span class="user">{{ username }}</span>
          <el-button link @click="logout">退出</el-button>
        </div>
      </el-header>
      <el-main>
        <router-view />
      </el-main>
    </el-container>
  </el-container>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { useRoute, useRouter } from 'vue-router'
import { Connection, Document, Link, Lock, Odometer, Setting, Switch } from '@element-plus/icons-vue'
import { api } from '../api'

const route = useRoute()
const router = useRouter()
const username = ref('')
const dark = ref(localStorage.getItem('gately-dark') === '1')

// 后续里程碑的模块先在菜单中占位，方便用户了解路线图
const planned = [
  { title: '动态域名', icon: Link },
  { title: 'STUN 穿透', icon: Connection },
  { title: 'Web 服务', icon: Document },
  { title: 'SSL 证书', icon: Lock },
]

function applyDark(v: string | number | boolean) {
  document.documentElement.classList.toggle('dark', Boolean(v))
  localStorage.setItem('gately-dark', v ? '1' : '0')
}

async function logout() {
  await api.post('/api/logout').catch(() => undefined)
  router.replace({ name: 'login' })
}

onMounted(async () => {
  applyDark(dark.value)
  const s = await api.get<{ username: string }>('/api/session')
  username.value = s.username
})
</script>

<style scoped>
.layout {
  height: 100%;
}
.aside {
  border-right: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.brand {
  height: 56px;
  display: flex;
  align-items: center;
  gap: 10px;
  padding: 0 20px;
  font-weight: 700;
  font-size: 18px;
}
.logo {
  width: 28px;
  height: 28px;
  border-radius: 7px;
  background: var(--gately-accent);
  color: #fff;
  display: inline-flex;
  align-items: center;
  justify-content: center;
  font-size: 16px;
}
.menu {
  border-right: none;
}
.soon {
  margin-left: auto;
}
.header {
  display: flex;
  align-items: center;
  justify-content: space-between;
  border-bottom: 1px solid var(--el-border-color-light);
  background: var(--el-bg-color);
}
.title {
  font-weight: 600;
}
.actions {
  display: flex;
  align-items: center;
  gap: 14px;
}
.user {
  color: var(--el-text-color-secondary);
}
</style>
