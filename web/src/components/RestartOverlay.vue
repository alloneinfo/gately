<template>
  <!-- 重启 / 恢复期间的全屏等待层：轮询 /api/session，服务恢复后自动刷新页面 -->
  <Teleport to="body">
    <div v-if="visible" class="restart-overlay">
      <div class="box">
        <template v-if="!timedOut">
          <Loader2 :size="36" class="spin" />
          <div class="title">{{ title }}</div>
          <div class="desc">服务恢复后页面会自动刷新，请勿关闭此页面。</div>
          <div class="elapsed">已等待 {{ elapsed }} 秒</div>
        </template>
        <template v-else>
          <TriangleAlert :size="36" class="warn" />
          <div class="title">等待超时</div>
          <div class="desc">服务长时间未响应。如修改了管理端口或监听地址，请手动访问新地址；也可以查看程序日志确认启动情况。</div>
          <el-button type="primary" :icon="RefreshCw" @click="reload">刷新页面</el-button>
        </template>
      </div>
    </div>
  </Teleport>
</template>

<script setup lang="ts">
import { onBeforeUnmount, ref } from 'vue'
import { Loader2, RefreshCw, TriangleAlert } from 'lucide-vue-next'

const visible = ref(false)
const title = ref('')
const elapsed = ref(0)
const timedOut = ref(false)
let timer: number | undefined
let ticker: number | undefined

function reload() {
  location.reload()
}

function stop() {
  window.clearTimeout(timer)
  window.clearInterval(ticker)
}

// start 显示等待层并开始轮询。为避免旧进程尚未退出时误判，须先观察到一次失败或等待满 6 秒才接受成功响应
function start(text: string) {
  stop()
  title.value = text
  visible.value = true
  timedOut.value = false
  elapsed.value = 0
  const begin = Date.now()
  let sawDown = false
  ticker = window.setInterval(() => {
    elapsed.value = Math.floor((Date.now() - begin) / 1000)
  }, 1000)
  const poll = async () => {
    if (Date.now() - begin > 120000) {
      timedOut.value = true
      stop()
      return
    }
    try {
      const resp = await fetch('/api/session', { credentials: 'same-origin', cache: 'no-store' })
      const ready = sawDown || Date.now() - begin > 6000
      if (ready && resp.status === 200) {
        stop()
        location.reload()
        return
      }
      if (ready && resp.status === 401) {
        // 恢复备份后账号可能变化，回到登录页
        stop()
        location.hash = '#/login'
        location.reload()
        return
      }
      if (resp.status >= 500) {
        sawDown = true
      }
    } catch {
      sawDown = true
    }
    timer = window.setTimeout(poll, 1500)
  }
  timer = window.setTimeout(poll, 1500)
}

onBeforeUnmount(stop)
defineExpose({ start })
</script>

<style scoped>
.restart-overlay {
  position: fixed;
  inset: 0;
  z-index: 5000;
  display: flex;
  align-items: center;
  justify-content: center;
  padding: 24px;
  background: color-mix(in srgb, var(--el-bg-color-page) 80%, transparent);
  backdrop-filter: blur(6px);
  -webkit-backdrop-filter: blur(6px);
}
.box {
  width: 100%;
  max-width: 380px;
  padding: 32px 28px;
  border-radius: 14px;
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
  text-align: center;
  display: flex;
  flex-direction: column;
  align-items: center;
  gap: 10px;
  box-shadow: 0 20px 50px -20px rgba(0, 0, 0, 0.3);
}
.spin {
  color: var(--el-color-primary);
  animation: spin 1s linear infinite;
}
.warn {
  color: var(--el-color-warning);
}
.title {
  font-size: 17px;
  font-weight: 600;
}
.desc {
  font-size: 13px;
  color: var(--el-text-color-secondary);
  line-height: 1.6;
}
.elapsed {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
@keyframes spin {
  to {
    transform: rotate(360deg);
  }
}
</style>
