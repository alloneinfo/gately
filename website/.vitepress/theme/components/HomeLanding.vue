<script setup lang="ts">
import { ref } from 'vue'
import { withBase } from 'vitepress'
import {
  Globe, RadioTower, ArrowLeftRight, ShieldCheck, Waypoints, ShieldAlert, Network, Boxes,
  Terminal, HardDrive, Clock, KeyRound, Copy, Check, Github, Lock, Eye, FileJson, Package,
  Bell, DatabaseBackup, Activity, Bot, UserCheck, ChartLine
} from 'lucide-vue-next'

const installTabs = [
  {
    key: 'docker',
    label: 'Docker',
    code: `docker run -d --name gately --network host \\
  -v ./gately-data:/etc/gately \\
  -v /var/run/docker.sock:/var/run/docker.sock \\
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep 初始密码`
  },
  {
    key: 'compose',
    label: 'Compose',
    code: `services:
  gately:
    image: ghcr.io/wangwanjie/gately:latest
    container_name: gately
    restart: unless-stopped
    network_mode: host
    volumes:
      - ./data:/etc/gately
      - /var/run/docker.sock:/var/run/docker.sock
    environment:
      - TZ=Asia/Shanghai`
  },
  {
    key: 'binary',
    label: '二进制',
    code: `# 从 Releases 下载对应平台的压缩包并解压
./gately -c /etc/gately

# 管理界面默认监听 :16680
# 用户名 admin，初始密码输出在启动日志中`
  }
]
const activeTab = ref('docker')
const copied = ref(false)

async function copyCode() {
  const tab = installTabs.find((t) => t.key === activeTab.value)
  if (!tab) {
    return
  }
  try {
    await navigator.clipboard.writeText(tab.code)
    copied.value = true
    setTimeout(() => {
      copied.value = false
    }, 1800)
  } catch {
    copied.value = false
  }
}

const stats = [
  { value: '~22 MB', label: '单文件，内嵌管理界面' },
  { value: '0', label: '默认连接项目方服务器' },
  { value: '全平台', label: 'Linux / macOS / Windows / FreeBSD，含 ARM、MIPS、RISC-V' },
  { value: 'Apache-2.0', label: '开源协议，源码可审计' }
]

const features = [
  { icon: Globe, title: '动态域名', text: '阿里云、Cloudflare、腾讯云、DNSPod、华为云、GoDaddy 等 11 家服务商；URL / 网卡 / 命令取 IP，IPv4 与 IPv6，Webhook 通知。' },
  { icon: RadioTower, title: 'STUN 穿透', text: 'TCP / UDP 打洞与保活，UPnP / NAT-PMP，内置转发；公网地址变化时触发 Webhook 与脚本。' },
  { icon: ArrowLeftRight, title: '端口转发', text: 'TCP / UDP、IPv4 / IPv6、端口段、多目标故障转移、连接数限制与流量统计。' },
  { icon: KeyRound, title: 'SSL 证书', text: 'ACME DNS-01 自动签发与续期，支持通配符与 EAB，可映射到目录并在变化后执行脚本。' },
  { icon: Waypoints, title: 'Web 服务', text: '按域名反向代理、重定向、静态文件；HTTPS、WebSocket、PROXY 协议；多后端权重与策略、健康检查、压缩、限流与响应缓存。' },
  { icon: ShieldAlert, title: 'WAF 防火墙', text: '内置 Coraza 与 OWASP CRS v4，按文件启用规则，支持自定义 SecLang，可拦截或仅检测。' },
  { icon: Network, title: '组网与穿透', text: '以进程托管方式集成 Tailscale、frp（frpc / frps）与 cloudflared 隧道，始终使用上游最新版本。' },
  { icon: Boxes, title: 'Docker 管理', text: '容器创建与编辑、日志与终端、Compose、镜像更新与一键升级（失败自动回滚）、网络与卷管理。' },
  { icon: Terminal, title: 'Web 终端', text: 'SSH、Telnet 与本机 Shell（默认关闭），支持代理、持久会话、SFTP 文件管理与分屏。' },
  { icon: HardDrive, title: '文件服务', text: 'WebDAV、FTP 服务与存储管理，支持本地与远程 WebDAV 挂载及在线文件管理。' },
  { icon: Clock, title: '计划任务与唤醒', text: 'cron 定时执行命令或 HTTP 请求并保留记录；网络唤醒魔术包广播与在线探测。' },
  { icon: ShieldCheck, title: '安全加固', text: '两步验证、登录失败限流、安全入口、仅内网访问、管理界面 HTTPS、API 令牌。' },
  { icon: UserCheck, title: '访问控制', text: '站点路径规则、临时分享链接、认证失败自动封禁、GeoIP 国家过滤与 IP 黑白名单。' },
  { icon: ChartLine, title: '访问统计与监控', text: '分钟级访问与流量统计，并以 Prometheus 指标暴露，便于接入已有监控。' },
  { icon: Bell, title: '通知中心', text: 'Webhook、Telegram、Bark、企业微信、钉钉、飞书与邮件；按级别与事件规则推送，支持静默去重。' },
  { icon: DatabaseBackup, title: '配置备份', text: '加密备份（scrypt + AES-GCM）、保留策略与定时备份，可写入本地目录或存储挂载点。' },
  { icon: Activity, title: '网络诊断', text: 'Ping、路由追踪、端口探测、DNS 查询、测速与局域网扫描，作为后台任务运行。' },
  { icon: Bot, title: 'API、命令行与 MCP', text: '多令牌按模块授权，自动生成的 OpenAPI 文档；gately ctl 调用实例，gately mcp 接入 AI 助手。' }
]

const promises = [
  { icon: Eye, title: '透明', text: '默认不连接任何项目方服务器，不上报设备信息。公网 IP 查询、STUN 服务器等所有对外地址都可见、可改、可关。' },
  { icon: FileJson, title: '可审计', text: '配置是明文 JSON，权限 0600，可直接纳入 git 做 diff 与回滚。敏感字段在 API 输出中脱敏。' },
  { icon: Lock, title: '自主可控', text: '第三方登录使用你自己申请的 OAuth 应用，不经过任何中转；不内置自更新，升级由 Docker 或包管理器负责。' },
  { icon: Package, title: '单文件部署', text: '一个静态二进制内嵌前端，支持 Docker、systemd、OpenWrt，覆盖 NAS、软路由与云主机。' }
]

const compare = [
  ['源码', '开源（Apache-2.0）', '闭源'],
  ['配置文件', '明文 JSON，可 git 管理', '专有格式（.lkcf）'],
  ['官方服务', '默认无，对外地址可改可关', '提供在线更新、第三方登录中转等官方服务'],
  ['第三方登录', '使用自己的 OAuth 应用', '经官方中转服务'],
  ['从 Lucky 迁移', '一键导入 .lkcf / 整站 zip', '—'],
  ['升级方式', 'Docker / 包管理器', '内置在线自更新'],
  ['功能广度', '覆盖常用核心能力', '更丰富，如 DLNA、网盘挂载等']
]

const faqs = [
  { q: 'Gately 与 Lucky 是什么关系？', a: 'Gately 是独立实现的开源项目，与 Lucky 项目方没有隶属关系，产品思路来自 Lucky，感谢其作者与社区的贡献。兼容性基于公开可观察的行为、接口与配置格式，不包含从 Lucky 二进制或前端产物复制的代码。' },
  { q: '可以直接迁移现有 Lucky 配置吗？', a: '可以。在「设置 → 备份与迁移」选择 .lkcf 文件或整站配置 zip，预览后导入。导入的规则默认停用，可与 Lucky 并存，逐个模块切换，也随时可以回退。' },
  { q: '需要联网到你们的服务器吗？', a: '不需要。Gately 默认不连接任何项目方服务器，也不会上报信息。' },
  { q: '适合在哪些设备上运行？', a: '家庭 NAS、软路由、云主机与各类服务器均可，提供 Linux（amd64 / arm64 / arm / mips / mipsle / riscv64）、macOS、Windows、FreeBSD 二进制，以及 amd64 / arm64 / arm/v7 的 Docker 镜像。' },
  { q: '端口转发、STUN 为什么推荐 host 网络？', a: '这些模块需要直接监听宿主机端口。不需要 Docker 管理时，可去掉 docker.sock 挂载。' }
]
const openFaq = ref(0)
</script>

<template>
  <div class="gl">
    <!-- Hero -->
    <section class="hero">
      <div class="wrap hero-grid">
        <div class="hero-text">
          <a class="pill" href="https://github.com/wangwanjie/gately/releases" target="_blank" rel="noreferrer">
            <span class="dot" />开源 · 可审计 · 可从 Lucky 一键迁移
          </a>
          <h1>看得见的<br /><span class="grad">网络网关工具集</span></h1>
          <p class="lead">
            DDNS、STUN 穿透、端口转发、反向代理、证书、WAF、Tailscale、frp，集成在一个约 22 MB 的单文件里。家庭网络、NAS、云主机与各类服务器均可使用。
          </p>
          <div class="cta">
            <a class="btn primary" :href="withBase('/docs/guide/quick-start')">快速开始</a>
            <a class="btn" :href="withBase('/download')">下载</a>
            <a class="btn ghost" href="https://github.com/wangwanjie/gately" target="_blank" rel="noreferrer">
              <Github :size="16" />GitHub
            </a>
          </div>
        </div>

        <div class="mock" aria-hidden="true">
          <div class="mock-bar"><i /><i /><i /><span>gately · 总览</span></div>
          <div class="mock-body">
            <div class="mock-row" v-for="r in [
              ['动态域名', '75 条记录已同步', 'ok'],
              ['STUN 穿透', '18 条规则运行中', 'ok'],
              ['端口转发', '25 条规则运行中', 'ok'],
              ['Web 服务', '27 个站点 · HTTPS', 'ok'],
              ['SSL 证书', '5 张证书 · 自动续期', 'ok'],
              ['WAF 防火墙', 'CRS v4 · 拦截模式', 'warn']
            ]" :key="r[0]">
              <span class="state" :class="r[2]" />
              <b>{{ r[0] }}</b>
              <em>{{ r[1] }}</em>
            </div>
          </div>
        </div>
      </div>
    </section>

    <!-- Stats -->
    <section class="wrap stats">
      <div v-for="s in stats" :key="s.label" class="stat">
        <strong>{{ s.value }}</strong>
        <span>{{ s.label }}</span>
      </div>
    </section>

    <!-- Features -->
    <section class="wrap block">
      <header class="head">
        <h2>常用网络能力，一处管理</h2>
        <p>覆盖家庭网络与服务器运维最常用的功能，每个模块都可单独启停。</p>
      </header>
      <div class="grid feat">
        <article v-for="f in features" :key="f.title" class="card">
          <div class="ico"><component :is="f.icon" :size="20" /></div>
          <h3>{{ f.title }}</h3>
          <p>{{ f.text }}</p>
        </article>
      </div>
      <p class="more"><a :href="withBase('/docs/modules/')">查看全部模块 →</a></p>
    </section>

    <!-- Promises -->
    <section class="band">
      <div class="wrap">
        <header class="head">
          <h2>运行在网络边界的软件，应该让你看得清楚</h2>
          <p>它手握 DNS 密钥与证书私钥，所以每一个行为都应当可以被检查。</p>
        </header>
        <div class="grid prom">
          <article v-for="p in promises" :key="p.title" class="card flat">
            <div class="ico"><component :is="p.icon" :size="20" /></div>
            <h3>{{ p.title }}</h3>
            <p>{{ p.text }}</p>
          </article>
        </div>
      </div>
    </section>

    <!-- Install -->
    <section class="wrap block">
      <header class="head">
        <h2>一分钟跑起来</h2>
        <p>镜像支持 amd64、arm64、arm/v7。启动后访问 <code>http://设备IP:16680</code>，用 <code>admin</code> 与日志中的初始密码登录。</p>
      </header>
      <div class="term">
        <div class="tabs">
          <button v-for="t in installTabs" :key="t.key" :class="{ on: activeTab === t.key }" @click="activeTab = t.key">
            {{ t.label }}
          </button>
          <button class="copy" @click="copyCode">
            <component :is="copied ? Check : Copy" :size="14" />{{ copied ? '已复制' : '复制' }}
          </button>
        </div>
        <pre><code>{{ installTabs.find((t) => t.key === activeTab)?.code }}</code></pre>
      </div>
      <p class="more"><a :href="withBase('/docs/guide/install')">完整安装文档 →</a></p>
    </section>

    <!-- Compare -->
    <section class="band">
      <div class="wrap">
        <header class="head">
          <h2>与 Lucky 对照</h2>
          <p>Gately 受 Lucky 启发，覆盖常用场景，并可一键导入现有配置。Lucky 的功能更丰富，两者可以并存使用。</p>
        </header>
        <div class="tbl">
          <table>
            <thead>
              <tr><th /><th>Gately</th><th>Lucky</th></tr>
            </thead>
            <tbody>
              <tr v-for="row in compare" :key="row[0]">
                <td>{{ row[0] }}</td>
                <td class="hl">{{ row[1] }}</td>
                <td>{{ row[2] }}</td>
              </tr>
            </tbody>
          </table>
        </div>
        <p class="more"><a :href="withBase('/docs/guide/migration')">阅读迁移指南 →</a></p>
      </div>
    </section>

    <!-- FAQ -->
    <section class="wrap block narrow">
      <header class="head"><h2>常见问题</h2></header>
      <div class="faq">
        <div v-for="(f, i) in faqs" :key="f.q" class="qa" :class="{ open: openFaq === i }">
          <button @click="openFaq = openFaq === i ? -1 : i">{{ f.q }}<span>{{ openFaq === i ? '−' : '+' }}</span></button>
          <p v-show="openFaq === i">{{ f.a }}</p>
        </div>
      </div>
      <p class="more"><a :href="withBase('/docs/guide/faq')">更多问题 →</a></p>
    </section>

    <!-- CTA -->
    <section class="wrap block">
      <div class="final">
        <h2>把网络边界的控制权拿回来</h2>
        <p>开源、透明、单文件。现在就部署一个 Gately。</p>
        <div class="cta center">
          <a class="btn primary lg" :href="withBase('/docs/guide/quick-start')">快速开始</a>
          <a class="btn lg" href="https://github.com/wangwanjie/gately" target="_blank" rel="noreferrer">在 GitHub 点亮 Star</a>
        </div>
      </div>
    </section>
  </div>
</template>

<style scoped>
.gl { color: var(--vp-c-text-1); overflow-x: hidden; }
.wrap { max-width: 1152px; margin: 0 auto; padding: 0 24px; }
.narrow { max-width: 820px; }
.block { padding-top: 72px; padding-bottom: 24px; }
.band { margin-top: 56px; padding: 64px 0 56px; background: var(--vp-c-bg-soft); border-block: 1px solid var(--vp-c-divider); }
.head { text-align: center; margin-bottom: 36px; }
.head h2 { font-size: 30px; font-weight: 700; letter-spacing: -0.01em; margin: 0; border: 0; padding: 0; }
.head p { margin: 12px auto 0; max-width: 640px; color: var(--vp-c-text-2); line-height: 1.7; }
.more { text-align: center; margin-top: 24px; }
.more a { color: var(--vp-c-brand-1); font-weight: 500; text-decoration: none; }
.more a:hover { text-decoration: underline; }

/* hero */
.hero { position: relative; padding: 72px 0 40px; background:
  radial-gradient(60% 60% at 15% 0%, var(--vp-c-brand-soft), transparent 70%),
  radial-gradient(40% 50% at 100% 20%, rgba(99, 102, 241, 0.1), transparent 70%); }
.hero-grid { display: grid; grid-template-columns: 1.05fr 1fr; gap: 48px; align-items: center; }
.pill { display: inline-flex; align-items: center; gap: 8px; padding: 6px 14px; border-radius: 999px; font-size: 13px; color: var(--vp-c-brand-1); background: var(--vp-c-brand-soft); text-decoration: none; }
.pill .dot { width: 7px; height: 7px; border-radius: 50%; background: var(--vp-c-brand-3); box-shadow: 0 0 0 4px var(--vp-c-brand-soft); }
h1 { font-size: 54px; line-height: 1.15; font-weight: 800; letter-spacing: -0.02em; margin: 20px 0 18px; }
.grad { background: linear-gradient(120deg, #18a888, #0b5f4f 55%, #6366f1); -webkit-background-clip: text; background-clip: text; color: transparent; }
.dark .grad { background: linear-gradient(120deg, #5eead4, #3ed1ae 50%, #a5b4fc); -webkit-background-clip: text; background-clip: text; }
.lead { font-size: 17px; line-height: 1.8; color: var(--vp-c-text-2); max-width: 540px; margin: 0; }
.cta { display: flex; flex-wrap: wrap; gap: 12px; margin-top: 28px; }
.cta.center { justify-content: center; }
.btn { display: inline-flex; align-items: center; gap: 6px; padding: 10px 22px; border-radius: 999px; font-weight: 600; font-size: 15px; text-decoration: none; color: var(--vp-c-text-1); background: var(--vp-c-bg-soft); border: 1px solid var(--vp-c-divider); transition: all .2s; }
.btn:hover { border-color: var(--vp-c-brand-1); color: var(--vp-c-brand-1); }
.btn.primary { color: #fff; background: var(--vp-c-brand-1); border-color: var(--vp-c-brand-1); }
.dark .btn.primary { color: #06261f; }
.btn.primary:hover { background: var(--vp-c-brand-2); color: #fff; }
.dark .btn.primary:hover { color: #06261f; }
.btn.ghost { background: transparent; }
.btn.lg { padding: 13px 30px; font-size: 16px; }

/* mock */
.mock { border-radius: 14px; background: var(--vp-c-bg); border: 1px solid var(--vp-c-divider); box-shadow: 0 24px 60px -20px rgba(11, 95, 79, .35); overflow: hidden; transform: perspective(1200px) rotateY(-4deg); }
.mock-bar { display: flex; align-items: center; gap: 6px; padding: 12px 14px; background: var(--vp-c-bg-soft); border-bottom: 1px solid var(--vp-c-divider); }
.mock-bar i { width: 10px; height: 10px; border-radius: 50%; background: var(--vp-c-divider); }
.mock-bar span { margin-left: 10px; font-size: 12px; color: var(--vp-c-text-3); font-family: var(--vp-font-family-mono); }
.mock-body { padding: 8px 6px 12px; }
.mock-row { display: grid; grid-template-columns: 14px 96px 1fr; align-items: center; gap: 10px; padding: 12px 14px; border-radius: 8px; font-size: 14px; }
.mock-row:nth-child(odd) { background: var(--vp-c-bg-soft); }
.mock-row em { font-style: normal; color: var(--vp-c-text-2); font-size: 13px; }
.state { width: 9px; height: 9px; border-radius: 50%; }
.state.ok { background: #22c55e; box-shadow: 0 0 0 3px rgba(34, 197, 94, .18); }
.state.warn { background: #f59e0b; box-shadow: 0 0 0 3px rgba(245, 158, 11, .18); }

/* stats */
.stats { display: grid; grid-template-columns: repeat(4, 1fr); gap: 16px; margin-top: 8px; }
.stat { padding: 22px 20px; border-radius: 14px; border: 1px solid var(--vp-c-divider); background: var(--vp-c-bg); text-align: center; }
.stat strong { display: block; font-size: 28px; font-weight: 800; color: var(--vp-c-brand-1); }
.stat span { display: block; margin-top: 6px; font-size: 13px; color: var(--vp-c-text-2); line-height: 1.5; }

/* cards */
.grid { display: grid; gap: 18px; }
.feat { grid-template-columns: repeat(3, 1fr); }
.prom { grid-template-columns: repeat(4, 1fr); }
.card { padding: 22px; border-radius: 14px; border: 1px solid var(--vp-c-divider); background: var(--vp-c-bg); transition: transform .2s, box-shadow .2s, border-color .2s; }
.card:hover { transform: translateY(-3px); border-color: var(--vp-c-brand-3); box-shadow: 0 12px 28px -14px rgba(11, 95, 79, .35); }
.card.flat { background: var(--vp-c-bg); }
.ico { display: grid; place-items: center; width: 38px; height: 38px; border-radius: 10px; color: var(--vp-c-brand-1); background: var(--vp-c-brand-soft); }
.card h3 { margin: 14px 0 8px; font-size: 17px; font-weight: 700; }
.card p { margin: 0; font-size: 14px; line-height: 1.7; color: var(--vp-c-text-2); }

/* terminal */
.term { max-width: 820px; margin: 0 auto; border-radius: 14px; overflow: hidden; background: #0f1b1a; border: 1px solid #1d3532; box-shadow: 0 18px 40px -22px rgba(0, 0, 0, .6); }
.tabs { display: flex; gap: 2px; padding: 8px 8px 0; background: #14211f; }
.tabs button { padding: 8px 16px; font-size: 13px; color: #8fa7a2; background: transparent; border: 0; border-radius: 8px 8px 0 0; cursor: pointer; }
.tabs button.on { color: #e6fffa; background: #0f1b1a; }
.tabs .copy { margin-left: auto; display: inline-flex; align-items: center; gap: 6px; border-radius: 8px; }
.tabs .copy:hover { color: #e6fffa; }
.term pre { margin: 0; padding: 20px 22px; overflow-x: auto; }
.term code { font-family: var(--vp-font-family-mono); font-size: 13.5px; line-height: 1.75; color: #c8f3e8; white-space: pre; }

/* compare */
.tbl { max-width: 860px; margin: 0 auto; border-radius: 14px; overflow: hidden; border: 1px solid var(--vp-c-divider); background: var(--vp-c-bg); }
.tbl table { width: 100%; border-collapse: collapse; display: table; margin: 0; }
.tbl th, .tbl td { padding: 13px 18px; text-align: left; font-size: 14px; border: 0; border-bottom: 1px solid var(--vp-c-divider); background: transparent !important; }
.tbl tr:last-child td { border-bottom: 0; }
.tbl th { color: var(--vp-c-text-2); font-weight: 600; }
.tbl td:first-child { color: var(--vp-c-text-2); width: 22%; }
.tbl td.hl { color: var(--vp-c-brand-1); font-weight: 600; }

/* faq */
.faq { border-top: 1px solid var(--vp-c-divider); }
.qa { border-bottom: 1px solid var(--vp-c-divider); }
.qa button { width: 100%; display: flex; justify-content: space-between; align-items: center; padding: 18px 4px; font-size: 16px; font-weight: 600; text-align: left; color: var(--vp-c-text-1); background: none; border: 0; cursor: pointer; }
.qa button span { font-size: 20px; color: var(--vp-c-brand-1); }
.qa p { margin: 0; padding: 0 4px 20px; line-height: 1.8; color: var(--vp-c-text-2); }

/* final */
.final { text-align: center; padding: 56px 24px; border-radius: 20px; border: 1px solid var(--vp-c-divider); background:
  radial-gradient(70% 120% at 50% 0%, var(--vp-c-brand-soft), transparent 70%), var(--vp-c-bg-soft); margin-bottom: 48px; }
.final h2 { margin: 0; font-size: 30px; font-weight: 800; border: 0; padding: 0; }
.final p { margin: 12px 0 0; color: var(--vp-c-text-2); }

@media (max-width: 960px) {
  .hero-grid { grid-template-columns: 1fr; }
  .mock { transform: none; }
  h1 { font-size: 40px; }
  .feat { grid-template-columns: repeat(2, 1fr); }
  .prom, .stats { grid-template-columns: repeat(2, 1fr); }
}
@media (max-width: 600px) {
  .feat, .prom, .stats { grid-template-columns: 1fr; }
  .head h2, .final h2 { font-size: 24px; }
}
</style>
