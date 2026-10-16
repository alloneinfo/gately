<template>
  <div>
    <div class="stats">
      <div class="stat">
        <div class="stat-icon"><Tag :size="18" /></div>
        <div class="stat-text">
          <div class="stat-label">版本</div>
          <div class="stat-value" :title="status ? `${status.commit} · ${status.buildDate}` : ''">{{ status?.version ?? '-' }}</div>
          <div class="stat-sub">{{ status ? `${status.os}/${status.arch} · ${status.goVersion}` : '' }}</div>
        </div>
      </div>
      <div class="stat">
        <div class="stat-icon"><Clock3 :size="18" /></div>
        <div class="stat-text">
          <div class="stat-label">已运行</div>
          <div class="stat-value">{{ uptime || '-' }}</div>
          <div class="stat-sub">{{ status ? '启动于 ' + new Date(status.startTime).toLocaleString() : '' }}</div>
        </div>
      </div>
      <div class="stat">
        <div class="stat-icon"><MemoryStick :size="18" /></div>
        <div class="stat-text">
          <div class="stat-label">内存占用</div>
          <div class="stat-value">{{ status ? formatBytes(status.memAlloc) : '-' }}</div>
          <div class="stat-sub">Go 堆内存</div>
        </div>
      </div>
      <div class="stat">
        <div class="stat-icon"><Activity :size="18" /></div>
        <div class="stat-text">
          <div class="stat-label">协程</div>
          <div class="stat-value">{{ status?.goroutines ?? '-' }}</div>
          <div class="stat-sub">goroutines</div>
        </div>
      </div>
    </div>

    <div class="section-title">功能模块</div>
    <div class="modules">
      <router-link v-for="m in moduleCards" :key="m.id" :to="m.nav.path" class="module" :class="`lv-${m.level}`">
        <div class="module-head">
          <div class="module-icon"><component :is="m.nav.icon" :size="20" /></div>
          <div class="module-titles">
            <div class="module-title">{{ m.nav.title }}</div>
            <div class="module-desc">{{ m.nav.desc }}</div>
          </div>
          <ChevronRight :size="16" class="module-arrow" />
        </div>
        <div v-if="m.summary.length" class="module-items">
          <div v-for="it in m.summary" :key="it.label" class="item">
            <span class="item-label">{{ it.label }}</span>
            <span :class="['item-value', it.level]">{{ it.value }}</span>
          </div>
        </div>
        <div v-else class="module-items empty">点击进入管理</div>
      </router-link>
    </div>

    <div class="bottom">
      <div class="g-card promise">
        <div class="g-card-head">
          <div class="g-card-title"><ShieldCheck :size="16" />透明度承诺</div>
        </div>
        <div class="g-card-body">
          <ul>
            <li><Check :size="14" />不连接任何项目方服务器，不上报设备信息</li>
            <li><Check :size="14" />所有对外地址（公网 IP 查询、STUN 服务器）都可见、可改</li>
            <li><Check :size="14" />配置为明文 JSON，可直接纳入 git 审计</li>
            <li><Check :size="14" />源码 Apache-2.0 开源，欢迎审查与贡献</li>
          </ul>
        </div>
      </div>
      <LogCard url="/api/logs" title="系统日志" class="logs" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { Activity, Check, ChevronRight, Clock3, MemoryStick, ShieldCheck, Tag } from 'lucide-vue-next'
import { api, formatBytes, type Status } from '../api'
import LogCard from '../components/LogCard.vue'
import { navItems, type NavItem } from '../nav'

const status = ref<Status>()
const now = ref(Date.now())
let timer: number | undefined

// 按导航顺序展示模块卡片，并取摘要中最严重的等级作为卡片颜色
const moduleCards = computed(() => {
  const mods = status.value?.modules ?? []
  const rank: Record<string, number> = { ok: 1, warn: 2, error: 3 }
  const out: { id: string; nav: NavItem; summary: { label: string; value: string; level?: string }[]; level: string }[] = []
  for (const nav of navItems) {
    if (!nav.moduleId) {
      continue
    }
    const m = mods.find((x) => x.id === nav.moduleId)
    if (!m) {
      continue
    }
    const summary = m.summary ?? []
    let level = 'none'
    for (const it of summary) {
      if (it.level && (rank[it.level] ?? 0) > (rank[level] ?? 0)) {
        level = it.level
      }
    }
    out.push({ id: m.id, nav, summary, level })
  }
  return out
})

const uptime = computed(() => {
  if (!status.value) {
    return ''
  }
  let s = Math.max(0, Math.floor((now.value - new Date(status.value.startTime).getTime()) / 1000))
  const d = Math.floor(s / 86400)
  s %= 86400
  const h = Math.floor(s / 3600)
  s %= 3600
  const m = Math.floor(s / 60)
  if (d > 0) {
    return `${d} 天 ${h} 小时`
  }
  if (h > 0) {
    return `${h} 小时 ${m} 分`
  }
  return `${m} 分钟`
})

async function load() {
  try {
    status.value = await api.get<Status>('/api/status')
    now.value = Date.now()
  } catch {
    // 下一轮重试
  }
}

onMounted(() => {
  load()
  timer = window.setInterval(load, 10000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.stats {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.stat {
  display: flex;
  gap: 12px;
  align-items: flex-start;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius);
  min-width: 0;
}
.stat-icon {
  width: 36px;
  height: 36px;
  border-radius: 9px;
  background: var(--el-color-primary-light-9);
  color: var(--el-color-primary);
  display: flex;
  align-items: center;
  justify-content: center;
  flex-shrink: 0;
}
.stat-text {
  min-width: 0;
}
.stat-label {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.stat-value {
  font-size: 18px;
  font-weight: 600;
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.stat-sub {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.section-title {
  margin: 24px 0 12px;
  font-size: 13px;
  font-weight: 600;
  color: var(--el-text-color-secondary);
}
.modules {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(260px, 1fr));
  gap: 12px;
}
.module {
  --lv: var(--el-color-primary);
  --lv-bg: var(--el-color-primary-light-9);
  display: flex;
  flex-direction: column;
  padding: 16px;
  background: var(--el-bg-color);
  border: 1px solid var(--g-border);
  border-radius: var(--g-radius);
  text-decoration: none;
  color: inherit;
  transition:
    border-color 0.15s,
    transform 0.15s;
  position: relative;
  overflow: hidden;
}
.module::before {
  content: '';
  position: absolute;
  inset: 0 auto 0 0;
  width: 3px;
  background: var(--lv);
  opacity: 0;
}
.module.lv-ok {
  --lv: var(--el-color-success);
  --lv-bg: var(--el-color-success-light-9);
}
.module.lv-warn {
  --lv: var(--el-color-warning);
  --lv-bg: var(--el-color-warning-light-9);
}
.module.lv-error {
  --lv: var(--el-color-danger);
  --lv-bg: var(--el-color-danger-light-9);
}
.module.lv-warn::before,
.module.lv-error::before {
  opacity: 1;
}
.module:hover {
  border-color: var(--el-color-primary-light-5);
}
.module-head {
  display: flex;
  align-items: center;
  gap: 12px;
}
.module-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--lv);
  background: var(--lv-bg);
  flex-shrink: 0;
}
.module-titles {
  flex: 1;
  min-width: 0;
}
.module-title {
  font-weight: 600;
  font-size: 15px;
}
.module-desc {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  margin-top: 2px;
  white-space: nowrap;
  overflow: hidden;
  text-overflow: ellipsis;
}
.module-arrow {
  color: var(--el-text-color-placeholder);
  flex-shrink: 0;
}
.module-items {
  margin-top: 14px;
  padding-top: 10px;
  border-top: 1px dashed var(--g-border);
}
.module-items.empty {
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}
.item {
  display: flex;
  justify-content: space-between;
  gap: 12px;
  font-size: 13px;
  line-height: 1.9;
}
.item-label {
  color: var(--el-text-color-secondary);
  flex-shrink: 0;
}
.item-value {
  font-weight: 500;
  text-align: right;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.item-value.ok {
  color: var(--el-color-success);
}
.item-value.warn {
  color: var(--el-color-warning);
}
.item-value.error {
  color: var(--el-color-danger);
}
.bottom {
  display: grid;
  grid-template-columns: 320px minmax(0, 1fr);
  gap: 12px;
  margin-top: 24px;
  align-items: start;
}
.bottom .logs {
  margin-top: 0;
}
.promise ul {
  margin: 0;
  padding: 0;
  list-style: none;
  font-size: 13px;
  line-height: 1.6;
  display: flex;
  flex-direction: column;
  gap: 10px;
}
.promise li {
  display: flex;
  gap: 8px;
  align-items: flex-start;
  color: var(--el-text-color-regular);
}
.promise li svg {
  color: var(--el-color-success);
  flex-shrink: 0;
  margin-top: 4px;
}
@media (max-width: 1100px) {
  .stats {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .bottom {
    grid-template-columns: 1fr;
  }
}
@media (max-width: 767.98px) {
  .stats {
    gap: 8px;
  }
  .stat {
    padding: 12px;
    gap: 10px;
  }
  .stat-icon {
    width: 30px;
    height: 30px;
  }
  .stat-value {
    font-size: 15px;
  }
  .modules {
    grid-template-columns: 1fr;
    gap: 8px;
  }
  .module {
    padding: 12px 14px;
  }
  .module-icon {
    width: 34px;
    height: 34px;
  }
  .module-items {
    margin-top: 10px;
    padding-top: 6px;
  }
  .module-items.empty {
    display: none;
  }
  .item {
    line-height: 1.75;
  }
  .section-title {
    margin-top: 18px;
  }
}
</style>
