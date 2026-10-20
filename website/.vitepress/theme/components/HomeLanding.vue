<script setup lang="ts">
import { computed, ref } from 'vue'
import { useData, withBase } from 'vitepress'
import AiNotice from './AiNotice.vue'
import {
  Globe, RadioTower, ArrowLeftRight, ShieldCheck, Waypoints, ShieldAlert, Network, Boxes,
  Terminal, HardDrive, Clock, KeyRound, Copy, Check, Github, Lock, Eye, FileJson, Package,
  Bell, DatabaseBackup, Activity, Bot, UserCheck, ChartLine
} from 'lucide-vue-next'

// 所有文案来自当前语言包（.vitepress/i18n/<语言>.json 的 home 节点），图标按顺序与条目一一对应
const { theme, localeIndex } = useData()
const h = computed(() => theme.value.home)
const prefix = computed(() => (localeIndex.value === 'root' ? '' : `/${localeIndex.value}`))
const link = (path: string) => withBase(`${prefix.value}${path}`)

const featureIcons = [
  Globe, RadioTower, ArrowLeftRight, KeyRound, Waypoints, ShieldAlert, Network, Boxes, Terminal,
  HardDrive, Clock, ShieldCheck, UserCheck, ChartLine, Bell, DatabaseBackup, Activity, Bot
]
const promiseIcons = [Eye, FileJson, Lock, Package]

const features = computed(() => h.value.features.map((f: any, i: number) => ({ ...f, icon: featureIcons[i] })))
const promises = computed(() => h.value.promises.map((f: any, i: number) => ({ ...f, icon: promiseIcons[i] })))

const COMPOSE = `services:
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

const installTabs = computed(() => [
  { key: 'docker', label: 'Docker', code: h.value.installDocker },
  { key: 'compose', label: 'Compose', code: COMPOSE },
  { key: 'binary', label: h.value.installTabBinary, code: h.value.installBinary }
])
const activeTab = ref('docker')
const copied = ref(false)

async function copyCode() {
  const tab = installTabs.value.find((x) => x.key === activeTab.value)
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

const openFaq = ref(0)
</script>

<template>
  <div class="gl">
    <div class="wrap notice-wrap"><AiNotice /></div>
    <!-- Hero -->
    <section class="hero">
      <div class="wrap hero-grid">
        <div class="hero-text">
          <!-- 有版本公告时链接到公告页，否则显示标语并链接到 GitHub Releases -->
          <a v-if="h.announce" class="pill" :href="h.announce.link">
            <span class="dot" />{{ h.announce.text }}
          </a>
          <a v-else class="pill" href="https://github.com/wangwanjie/gately/releases" target="_blank" rel="noreferrer">
            <span class="dot" />{{ h.pill }}
          </a>
          <h1>{{ h.titleLine1 }}<br /><span class="grad">{{ h.titleLine2 }}</span></h1>
          <p class="lead">{{ h.lead }}</p>
          <div class="cta">
            <a class="btn primary" :href="link('/docs/guide/quick-start')">{{ h.ctaStart }}</a>
            <a class="btn" :href="link('/download')">{{ h.ctaDownload }}</a>
            <a class="btn ghost" href="https://github.com/wangwanjie/gately" target="_blank" rel="noreferrer">
              <Github :size="16" />GitHub
            </a>
          </div>
        </div>

        <div class="mock" aria-hidden="true">
          <div class="mock-bar"><i /><i /><i /><span>{{ h.mockTitle }}</span></div>
          <div class="mock-body">
            <div class="mock-row" v-for="r in h.mockRows" :key="r[0]">
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
      <div v-for="s in h.stats" :key="s.label" class="stat">
        <strong>{{ s.value }}</strong>
        <span>{{ s.label }}</span>
      </div>
    </section>

    <!-- Features -->
    <section class="wrap block">
      <header class="head">
        <h2>{{ h.featuresTitle }}</h2>
        <p>{{ h.featuresDesc }}</p>
      </header>
      <div class="grid feat">
        <article v-for="f in features" :key="f.title" class="card">
          <div class="ico"><component :is="f.icon" :size="20" /></div>
          <h3>{{ f.title }}</h3>
          <p>{{ f.text }}</p>
        </article>
      </div>
      <p class="more"><a :href="link('/docs/modules/')">{{ h.moreModules }}</a></p>
    </section>

    <!-- Promises -->
    <section class="band">
      <div class="wrap">
        <header class="head">
          <h2>{{ h.promisesTitle }}</h2>
          <p>{{ h.promisesDesc }}</p>
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
        <h2>{{ h.installTitle }}</h2>
        <!-- eslint-disable-next-line vue/no-v-html -->
        <p v-html="h.installDesc" />
      </header>
      <div class="term">
        <div class="tabs">
          <button v-for="tab in installTabs" :key="tab.key" :class="{ on: activeTab === tab.key }" @click="activeTab = tab.key">
            {{ tab.label }}
          </button>
          <button class="copy" @click="copyCode">
            <component :is="copied ? Check : Copy" :size="14" />{{ copied ? h.copied : h.copy }}
          </button>
        </div>
        <pre><code>{{ installTabs.find((x) => x.key === activeTab)?.code }}</code></pre>
      </div>
      <p class="more"><a :href="link('/docs/guide/install')">{{ h.fullInstall }}</a></p>
    </section>

    <!-- FAQ -->
    <section class="wrap block narrow">
      <header class="head"><h2>{{ h.faqTitle }}</h2></header>
      <div class="faq">
        <div v-for="(f, i) in h.faqs" :key="f.q" class="qa" :class="{ open: openFaq === i }">
          <button @click="openFaq = openFaq === i ? -1 : i">{{ f.q }}<span>{{ openFaq === i ? '−' : '+' }}</span></button>
          <p v-show="openFaq === i">{{ f.a }}</p>
        </div>
      </div>
      <p class="more"><a :href="link('/docs/guide/faq')">{{ h.moreFaq }}</a></p>
    </section>

    <!-- CTA -->
    <section class="wrap block">
      <div class="final">
        <h2>{{ h.finalTitle }}</h2>
        <p>{{ h.finalDesc }}</p>
        <div class="cta center">
          <a class="btn primary lg" :href="link('/docs/guide/quick-start')">{{ h.ctaStart }}</a>
          <a class="btn lg" href="https://github.com/wangwanjie/gately" target="_blank" rel="noreferrer">{{ h.starOnGithub }}</a>
        </div>
        <!-- Telegram 群二维码：手机扫码即可加入 -->
        <a class="tg" href="https://t.me/gately_chat" target="_blank" rel="noreferrer">
          <img class="tg-qr" :src="withBase('/telegram-qr.svg')" :alt="h.telegramScan" width="128" height="128" loading="lazy" />
          <span>{{ h.telegramScan }}</span>
        </a>
      </div>
    </section>
  </div>
</template>

<style scoped>
.notice-wrap { padding-top: 16px; padding-bottom: 0; }
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
.tg { display: inline-flex; flex-direction: column; align-items: center; gap: 10px; margin-top: 32px; color: var(--vp-c-text-2); font-size: 14px; text-decoration: none; }
.tg:hover { color: var(--vp-c-brand-1); }
.tg-qr { display: block; border-radius: 12px; border: 1px solid var(--vp-c-divider); background: #fff; }

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
