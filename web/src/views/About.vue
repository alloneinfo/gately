<template>
  <div class="about">
    <!-- ============ 项目头部 ============ -->
    <section class="hero">
      <div class="hero-glow" />
      <AppLogo :size="76" class="hero-logo" />
      <h1>Gately</h1>
      <p class="tagline">开源、可审计的家庭网络网关</p>
      <p class="intro">
        DDNS、STUN 穿透、端口转发、反向代理、SSL 证书、WAF 等常用能力集成在一个单文件中，可一键导入 Lucky 配置。每一行代码都公开，每一个对外地址都可见。
      </p>
      <div class="badges">
        <span class="badge"><span class="bk">version</span><span class="bv mono">{{ info?.version ?? '-' }}</span></span>
        <span class="badge"><span class="bk">license</span><span class="bv">Apache-2.0</span></span>
        <span class="badge"><span class="bk">go</span><span class="bv mono">{{ info?.goVersion?.replace(/^go/, '') ?? '-' }}</span></span>
        <span class="badge"><span class="bk">platform</span><span class="bv mono">{{ info ? `${info.os}/${info.arch}` : '-' }}</span></span>
        <span class="badge"><span class="bk">docker</span><span class="bv">amd64 · arm64 · arm/v7</span></span>
      </div>
      <div class="actions">
        <el-button type="primary" tag="a" :href="REPO" target="_blank" rel="noopener" :icon="Github">GitHub 仓库</el-button>
        <el-button tag="a" :href="`${REPO}/releases`" target="_blank" rel="noopener" :icon="Package">版本发布</el-button>
        <el-button tag="a" :href="`${REPO}/issues`" target="_blank" rel="noopener" :icon="MessageSquare">反馈问题</el-button>
        <el-button :icon="RefreshCw" :loading="checking" @click="checkUpdate">检查更新</el-button>
      </div>
      <el-alert v-if="update" :type="update.hasUpdate ? 'success' : 'info'" :closable="false" show-icon class="update">
        <template #title>
          <template v-if="update.hasUpdate">发现新版本 {{ update.latest }}（当前 {{ update.current }}）</template>
          <template v-else>当前已是最新版本{{ update.latest ? `（最新发布 ${update.latest}）` : '' }}</template>
        </template>
        <template #default>
          <div v-if="update.publishedAt && update.latest" class="meta">发布于 {{ new Date(update.publishedAt).toLocaleString() }}</div>
          <pre v-if="update.hasUpdate && update.notes" class="notes">{{ update.notes }}</pre>
          <a v-if="update.url" :href="update.url" target="_blank" rel="noopener">查看发布页</a>
        </template>
      </el-alert>
      <div class="hint">检查更新只在点击时访问 api.github.com，平时不会主动联网。</div>
    </section>

    <!-- ============ 为什么 ============ -->
    <section class="block">
      <h2><Sparkles :size="18" />为什么做 Gately</h2>
      <p class="lead">
        Lucky 功能强大，但闭源、二进制加壳，配置文件经过混淆，并包含与官方服务器通信的逻辑。对于运行在网络边界、手握 DNS 密钥和证书私钥的软件，用户应当能看到它在做什么。Gately
        用开源代码重新实现了这些能力，并保持配置可迁移，让你在不改变使用习惯的前提下换到一个可以审计的实现。
      </p>
    </section>

    <!-- ============ 透明度承诺 ============ -->
    <section class="block">
      <h2><ShieldCheck :size="18" />透明度承诺</h2>
      <div class="promises">
        <div v-for="p in promises" :key="p.title" class="promise">
          <div class="promise-icon"><component :is="p.icon" :size="20" /></div>
          <div class="promise-title">{{ p.title }}</div>
          <div class="promise-desc">{{ p.desc }}</div>
        </div>
      </div>
    </section>

    <!-- ============ 功能 ============ -->
    <section class="block">
      <h2><LayoutGrid :size="18" />功能一览</h2>
      <div class="features">
        <div v-for="f in features" :key="f.title" class="feature">
          <div class="feature-head">
            <div class="feature-icon"><component :is="f.icon" :size="18" /></div>
            <div class="feature-title">{{ f.title }}</div>
          </div>
          <div class="feature-tags">
            <span v-for="t in f.tags" :key="t">{{ t }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- ============ 快速开始 ============ -->
    <section class="block two">
      <div>
        <h2><Rocket :size="18" />快速开始</h2>
        <p class="muted small">Docker 镜像支持 amd64、arm64、arm/v7，推荐使用 host 网络以便监听宿主机端口。</p>
        <div class="code">
          <button class="code-copy" title="复制" @click="copy(DOCKER_CMD)"><Copy :size="14" /></button>
          <pre class="mono">{{ DOCKER_CMD }}</pre>
        </div>
        <p class="muted small">
          也可以从
          <a :href="`${REPO}/releases`" target="_blank" rel="noopener">Releases</a>
          下载 Linux / macOS / Windows / FreeBSD 二进制，执行 <code class="mono">./gately -c /etc/gately</code> 运行。
        </p>
      </div>
      <div>
        <h2><ArrowRightLeft :size="18" />从 Lucky 迁移</h2>
        <ol class="steps">
          <li>
            <b>导出</b>
            <span>找到 Lucky 配置目录中的 <code class="mono">lucky_*.lkcf</code>，或整站配置 zip。</span>
          </li>
          <li>
            <b>导入</b>
            <span>在 <router-link to="/settings?tab=backup">设置 → 备份与迁移</router-link> 中上传，预览后导入，规则默认停用。</span>
          </li>
          <li>
            <b>切换</b>
            <span>按模块逐个在 Lucky 中停用、在 Gately 中启用，确认无误后再停掉 Lucky。</span>
          </li>
        </ol>
        <a :href="`${REPO}/blob/main/docs/MIGRATION.md`" target="_blank" rel="noopener" class="link"><BookOpen :size="14" />完整迁移指南</a>
      </div>
    </section>

    <!-- ============ 底部 ============ -->
    <footer class="foot">
      <div class="stack">
        <span v-for="s in stack" :key="s">{{ s }}</span>
      </div>
      <div class="muted small">
        以 <a :href="`${REPO}/blob/main/LICENSE`" target="_blank" rel="noopener">Apache License 2.0</a> 发布 · 感谢
        <a href="https://github.com/gdy666/lucky" target="_blank" rel="noopener">Lucky</a> 带来的产品灵感 · 与 Lucky 的兼容仅基于公开可观察的行为与配置格式
      </div>
      <div v-if="info" class="muted small mono">{{ info.version }} · {{ info.commit || 'none' }} · {{ info.buildDate || 'unknown' }}</div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import {
  ArrowLeftRight,
  ArrowRightLeft,
  BookOpen,
  CalendarClock,
  Container,
  Copy,
  Eye,
  FileBraces,
  FolderSync,
  Github,
  Globe,
  Import,
  LayoutGrid,
  LockKeyhole,
  MessageSquare,
  Package,
  Power,
  Radar,
  RefreshCw,
  Rocket,
  Scale,
  Server,
  ShieldCheck,
  ShieldHalf,
  Sparkles,
  SquareTerminal,
  WifiOff,
} from 'lucide-vue-next'
import { api, type SystemInfo, type UpdateInfo } from '../api'
import AppLogo from '../components/AppLogo.vue'

const REPO = 'https://github.com/wangwanjie/gately'
const DOCKER_CMD = `docker run -d --name gately --network host \\
  -v ./gately-data:/etc/gately \\
  -v /var/run/docker.sock:/var/run/docker.sock \\
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep 初始密码`

const promises = [
  { icon: Scale, title: '代码开源', desc: '源码以 Apache-2.0 开源，任何人都可以审查、构建和贡献。' },
  { icon: WifiOff, title: '不回传数据', desc: '不连接任何项目方服务器，不上报设备信息；第三方登录使用你自己的 OAuth 应用。' },
  { icon: Eye, title: '对外地址可见', desc: '公网 IP 查询、STUN 服务器等所有对外地址都可见、可改、可关。' },
  { icon: FileBraces, title: '配置明文', desc: '配置为明文 JSON，可直接纳入 git 做审计和回滚。' },
]

// 与 README 的功能表保持一致
const features = [
  { icon: Globe, title: '动态域名', tags: ['阿里云', 'Cloudflare', '腾讯云 / DNSPod', '华为云', 'GoDaddy', 'Porkbun', 'dynv6', 'DuckDNS', '自定义回调', 'IPv4 / IPv6', 'Webhook'] },
  { icon: Radar, title: 'STUN 穿透', tags: ['TCP / UDP', '保活', 'UPnP / NAT-PMP', '内置转发', '地址变化通知', '脚本'] },
  { icon: ArrowLeftRight, title: '端口转发', tags: ['TCP / UDP', 'IPv4 / IPv6', '端口段', '故障转移', '连接数限制', '流量统计'] },
  { icon: ShieldCheck, title: 'SSL 证书', tags: ['ACME DNS-01', '通配符', 'EAB', '自动续期', '上传 / 读取文件', '映射到目录'] },
  { icon: Server, title: 'Web 服务', tags: ['反向代理', '重定向', '静态文件', 'HTTPS', 'WebSocket', 'PROXY 协议', 'OIDC / GitHub 登录', 'IP 黑白名单'] },
  { icon: ShieldHalf, title: 'WAF 防火墙', tags: ['Coraza', 'OWASP CRS v4', '按文件启用规则', '自定义 SecLang', '仅检测模式'] },
  { icon: Container, title: 'Docker 管理', tags: ['容器启停', '日志', '镜像更新检查', '一键升级', '失败自动回滚'] },
  { icon: SquareTerminal, title: 'Web 终端', tags: ['SSH', '主机密钥信任', '本机 Shell'] },
  { icon: Power, title: '网络唤醒', tags: ['魔术包广播', '在线探测'] },
  { icon: FolderSync, title: 'WebDAV', tags: ['多用户', '独立目录', '只读'] },
  { icon: CalendarClock, title: '计划任务', tags: ['cron 表达式', '命令', 'HTTP 请求', '执行记录'] },
  { icon: LockKeyhole, title: '安全与设置', tags: ['两步验证', '登录限流', '安全入口', '仅内网访问', 'API 令牌', 'DoH', '备份恢复'] },
  { icon: Import, title: '从 Lucky 迁移', tags: ['.lkcf', '整站 zip', '端口转发', 'DDNS', 'STUN', '证书', '反向代理', 'WAF'] },
]

const stack = ['Go', 'Vue 3', 'TypeScript', 'Element Plus', 'lego', 'Coraza', 'go:embed 单文件']

const info = ref<SystemInfo>()
const update = ref<UpdateInfo>()
const checking = ref(false)

async function checkUpdate() {
  checking.value = true
  try {
    update.value = await api.get<UpdateInfo>('/api/system/update')
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    checking.value = false
  }
}

async function copy(text: string) {
  try {
    await navigator.clipboard.writeText(text)
    ElMessage.success('已复制')
  } catch {
    ElMessage.warning('复制失败，请手动选择')
  }
}

onMounted(async () => {
  try {
    info.value = await api.get<SystemInfo>('/api/system/info')
  } catch {
    // 版本信息缺失时徽章显示为“-”
  }
})
</script>

<style scoped>
.about {
  max-width: 1080px;
  margin: 0 auto;
}

/* ---------- 头部 ---------- */
.hero {
  position: relative;
  overflow: hidden;
  text-align: center;
  padding: 48px 24px 32px;
  border-radius: 16px;
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
}
.hero-glow {
  position: absolute;
  inset: -40% -10% auto;
  height: 360px;
  background:
    radial-gradient(40% 60% at 35% 50%, color-mix(in srgb, var(--el-color-primary) 22%, transparent), transparent 70%),
    radial-gradient(35% 55% at 68% 45%, color-mix(in srgb, var(--g-chart-2) 18%, transparent), transparent 70%);
  filter: blur(10px);
  pointer-events: none;
}
.hero > *:not(.hero-glow) {
  position: relative;
}
.hero-logo {
  margin: 0 auto;
  filter: drop-shadow(0 10px 20px color-mix(in srgb, var(--el-color-primary) 35%, transparent));
}
h1 {
  margin: 18px 0 4px;
  font-size: 40px;
  font-weight: 800;
  letter-spacing: -0.5px;
  background: linear-gradient(135deg, var(--el-color-primary), var(--g-chart-2));
  -webkit-background-clip: text;
  background-clip: text;
  color: transparent;
}
.tagline {
  margin: 0;
  font-size: 17px;
  font-weight: 600;
}
.intro {
  max-width: 640px;
  margin: 12px auto 0;
  font-size: 14px;
  line-height: 1.8;
  color: var(--el-text-color-regular);
}
.badges {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 8px;
  margin-top: 20px;
}
.badge {
  display: inline-flex;
  font-size: 12px;
  line-height: 22px;
  border-radius: 5px;
  overflow: hidden;
  border: 1px solid var(--g-border);
}
.bk {
  padding: 0 8px;
  color: var(--el-text-color-secondary);
  background: var(--el-fill-color-light);
}
.bv {
  padding: 0 8px;
  color: #fff;
  background: var(--el-color-primary);
}
.actions {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 24px;
}
.actions .el-button + .el-button {
  margin-left: 0;
}
.actions a.el-button {
  text-decoration: none;
}
.update {
  max-width: 560px;
  margin: 20px auto 0;
  text-align: left;
}
.notes {
  white-space: pre-wrap;
  font-family: inherit;
  font-size: 12px;
  margin: 6px 0;
  max-height: 200px;
  overflow: auto;
}
.hint {
  margin-top: 14px;
  font-size: 12px;
  color: var(--el-text-color-placeholder);
}

/* ---------- 内容区块 ---------- */
.block {
  margin-top: 36px;
}
h2 {
  display: flex;
  align-items: center;
  gap: 8px;
  margin: 0 0 14px;
  padding-bottom: 10px;
  font-size: 18px;
  font-weight: 700;
  border-bottom: 1px solid var(--g-border);
}
h2 svg {
  color: var(--el-color-primary);
}
.lead {
  margin: 0;
  font-size: 14px;
  line-height: 1.9;
  color: var(--el-text-color-regular);
}
.small {
  font-size: 13px;
  line-height: 1.7;
}
code {
  font-size: 12px;
  padding: 1px 6px;
  border-radius: 4px;
  background: var(--el-fill-color);
}
.promises {
  display: grid;
  grid-template-columns: repeat(4, minmax(0, 1fr));
  gap: 12px;
}
.promise {
  padding: 18px;
  border-radius: var(--g-radius);
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
}
.promise-icon {
  width: 40px;
  height: 40px;
  border-radius: 10px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-success);
  background: var(--el-color-success-light-9);
}
.promise-title {
  margin-top: 12px;
  font-weight: 600;
  font-size: 15px;
}
.promise-desc {
  margin-top: 6px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-secondary);
}
.features {
  display: grid;
  grid-template-columns: repeat(auto-fill, minmax(300px, 1fr));
  gap: 12px;
}
.feature {
  padding: 16px;
  border-radius: var(--g-radius);
  border: 1px solid var(--g-border);
  background: var(--el-bg-color);
  transition:
    border-color 0.15s,
    transform 0.15s;
}
.feature:hover {
  border-color: var(--el-color-primary-light-5);
  transform: translateY(-1px);
}
.feature-head {
  display: flex;
  align-items: center;
  gap: 10px;
}
.feature-icon {
  width: 34px;
  height: 34px;
  border-radius: 9px;
  display: flex;
  align-items: center;
  justify-content: center;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.feature-title {
  font-weight: 600;
  font-size: 15px;
}
.feature-tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
  margin-top: 12px;
}
.feature-tags span {
  font-size: 12px;
  line-height: 20px;
  padding: 0 8px;
  border-radius: 999px;
  color: var(--el-text-color-regular);
  background: var(--el-fill-color-light);
}
.two {
  display: grid;
  grid-template-columns: minmax(0, 1.2fr) minmax(0, 1fr);
  gap: 32px;
}
.code {
  position: relative;
  margin: 10px 0;
  border-radius: var(--g-radius);
  background: #0f1720;
  border: 1px solid var(--g-border);
}
.code pre {
  margin: 0;
  padding: 14px 16px;
  overflow-x: auto;
  font-size: 12.5px;
  line-height: 1.7;
  color: #d4e4dc;
}
.code-copy {
  position: absolute;
  top: 8px;
  right: 8px;
  width: 28px;
  height: 28px;
  border: none;
  border-radius: 6px;
  cursor: pointer;
  display: flex;
  align-items: center;
  justify-content: center;
  color: #9fb3aa;
  background: rgba(255, 255, 255, 0.06);
}
.code-copy:hover {
  color: #fff;
  background: rgba(255, 255, 255, 0.14);
}
.steps {
  margin: 0;
  padding: 0;
  list-style: none;
  counter-reset: step;
}
.steps li {
  position: relative;
  padding: 0 0 16px 40px;
  font-size: 13px;
  line-height: 1.7;
  counter-increment: step;
}
.steps li::before {
  content: counter(step);
  position: absolute;
  left: 0;
  top: 0;
  width: 26px;
  height: 26px;
  border-radius: 50%;
  display: flex;
  align-items: center;
  justify-content: center;
  font-size: 12px;
  font-weight: 700;
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.steps li:not(:last-child)::after {
  content: '';
  position: absolute;
  left: 12.5px;
  top: 30px;
  bottom: 4px;
  width: 1px;
  background: var(--g-border);
}
.steps b {
  display: block;
  font-size: 14px;
}
.steps span {
  color: var(--el-text-color-secondary);
}
.link {
  display: inline-flex;
  align-items: center;
  gap: 4px;
  font-size: 13px;
  text-decoration: none;
}
.foot {
  margin: 40px 0 8px;
  padding-top: 20px;
  border-top: 1px solid var(--g-border);
  text-align: center;
  display: flex;
  flex-direction: column;
  gap: 8px;
}
.stack {
  display: flex;
  justify-content: center;
  flex-wrap: wrap;
  gap: 6px;
}
.stack span {
  font-size: 12px;
  padding: 2px 10px;
  border-radius: 999px;
  border: 1px solid var(--g-border);
  color: var(--el-text-color-secondary);
}

@media (max-width: 1000px) {
  .promises {
    grid-template-columns: repeat(2, minmax(0, 1fr));
  }
  .two {
    grid-template-columns: 1fr;
    gap: 28px;
  }
}
@media (max-width: 767.98px) {
  .hero {
    padding: 32px 16px 24px;
  }
  h1 {
    font-size: 32px;
  }
  .intro {
    font-size: 13px;
  }
  .actions .el-button {
    flex: 1 1 40%;
  }
  .block {
    margin-top: 28px;
  }
  .promises {
    grid-template-columns: 1fr;
  }
  .features {
    grid-template-columns: 1fr;
  }
}
</style>
