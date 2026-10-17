<template>
  <div class="about">
    <!-- ============ 项目头部 ============ -->
    <section class="hero">
      <div class="hero-glow" />
      <AppLogo :size="76" class="hero-logo" />
      <h1>Gately</h1>
      <p class="tagline">{{ t('about.tagline') }}</p>
      <p class="intro">
        {{ t('about.intro') }}
      </p>
      <div class="badges">
        <span class="badge"><span class="bk">version</span><span class="bv mono">{{ info?.version ?? '-' }}</span></span>
        <span class="badge"><span class="bk">license</span><span class="bv">Apache-2.0</span></span>
        <span class="badge"><span class="bk">go</span><span class="bv mono">{{ info?.goVersion?.replace(/^go/, '') ?? '-' }}</span></span>
        <span class="badge"><span class="bk">platform</span><span class="bv mono">{{ info ? `${info.os}/${info.arch}` : '-' }}</span></span>
        <span class="badge"><span class="bk">docker</span><span class="bv">amd64 · arm64 · arm/v7</span></span>
      </div>
      <div class="actions">
        <el-button type="primary" tag="a" :href="REPO" target="_blank" rel="noopener" :icon="Github">{{ t('about.repo') }}</el-button>
        <el-button tag="a" :href="`${REPO}/releases`" target="_blank" rel="noopener" :icon="Package">{{ t('about.releases') }}</el-button>
        <el-button tag="a" :href="`${REPO}/issues`" target="_blank" rel="noopener" :icon="MessageSquare">{{ t('about.feedback') }}</el-button>
        <el-button :icon="RefreshCw" :loading="checking" @click="checkUpdate">{{ t('about.checkUpdate') }}</el-button>
      </div>
      <el-alert v-if="update" :type="update.hasUpdate ? 'success' : 'info'" :closable="false" show-icon class="update">
        <template #title>
          <template v-if="update.hasUpdate">{{ t('about.newVersion', { latest: update.latest, current: update.current }) }}</template>
          <template v-else>{{ t('about.upToDate') }}{{ update.latest ? t('about.latestRelease', { latest: update.latest }) : '' }}</template>
        </template>
        <template #default>
          <div v-if="update.publishedAt && update.latest" class="meta">{{ t('about.publishedAt', { time: new Date(update.publishedAt).toLocaleString() }) }}</div>
          <pre v-if="update.hasUpdate && update.notes" class="notes">{{ update.notes }}</pre>
          <a v-if="update.url" :href="update.url" target="_blank" rel="noopener">{{ t('about.viewRelease') }}</a>
        </template>
      </el-alert>
      <div class="hint">{{ t('about.updateHint') }}</div>
    </section>

    <!-- ============ 为什么 ============ -->
    <section class="block">
      <h2><Sparkles :size="18" />{{ t('about.whyTitle') }}</h2>
      <p class="lead">{{ t('about.whyBody') }}</p>
    </section>

    <!-- ============ 透明度承诺 ============ -->
    <section class="block">
      <h2><ShieldCheck :size="18" />{{ t('about.promiseTitle') }}</h2>
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
      <h2><LayoutGrid :size="18" />{{ t('about.featuresTitle') }}</h2>
      <div class="features">
        <div v-for="f in features" :key="f.title" class="feature">
          <div class="feature-head">
            <div class="feature-icon"><component :is="f.icon" :size="18" /></div>
            <div class="feature-title">{{ f.title }}</div>
          </div>
          <div class="feature-tags">
            <span v-for="tag in f.tags" :key="tag">{{ tag }}</span>
          </div>
        </div>
      </div>
    </section>

    <!-- ============ 快速开始 ============ -->
    <section class="block two">
      <div>
        <h2><Rocket :size="18" />{{ t('about.quickStart') }}</h2>
        <p class="muted small">{{ t('about.dockerNote') }}</p>
        <div class="code">
          <button class="code-copy" :title="t('common.copy')" @click="copy(DOCKER_CMD)"><Copy :size="14" /></button>
          <pre class="mono">{{ DOCKER_CMD }}</pre>
        </div>
        <p class="muted small">
          <i18n-t keypath="about.binaryNote" tag="span">
            <template #releases>
              <a :href="`${REPO}/releases`" target="_blank" rel="noopener">Releases</a>
            </template>
            <template #cmd><code class="mono">./gately -c /etc/gately</code></template>
          </i18n-t>
        </p>
      </div>
      <div>
        <h2><ArrowRightLeft :size="18" />{{ t('about.migrateTitle') }}</h2>
        <ol class="steps">
          <li>
            <b>{{ t('about.stepExport') }}</b>
            <span>
              <i18n-t keypath="about.stepExportDesc" tag="span">
                <template #file><code class="mono">lucky_*.lkcf</code></template>
              </i18n-t>
            </span>
          </li>
          <li>
            <b>{{ t('about.stepImport') }}</b>
            <span>
              <i18n-t keypath="about.stepImportDesc" tag="span">
                <template #link><router-link to="/settings?tab=backup">{{ t('about.backupLink') }}</router-link></template>
              </i18n-t>
            </span>
          </li>
          <li>
            <b>{{ t('about.stepSwitch') }}</b>
            <span>{{ t('about.stepSwitchDesc') }}</span>
          </li>
        </ol>
        <a :href="`${REPO}/blob/main/docs/MIGRATION.md`" target="_blank" rel="noopener" class="link"><BookOpen :size="14" />{{ t('about.migrationGuide') }}</a>
      </div>
    </section>

    <!-- ============ 底部 ============ -->
    <footer class="foot">
      <div class="stack">
        <span v-for="s in stack" :key="s">{{ s }}</span>
      </div>
      <div class="muted small">
        <i18n-t keypath="about.footerNote" tag="span">
          <template #license><a :href="`${REPO}/blob/main/LICENSE`" target="_blank" rel="noopener">Apache License 2.0</a></template>
          <template #lucky><a href="https://github.com/gdy666/lucky" target="_blank" rel="noopener">Lucky</a></template>
        </i18n-t>
      </div>
      <div v-if="info" class="muted small mono">{{ info.version }} · {{ info.commit || 'none' }} · {{ info.buildDate || 'unknown' }}</div>
    </footer>
  </div>
</template>

<script setup lang="ts">
import { computed, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
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

const { t } = useI18n()

const REPO = 'https://github.com/wangwanjie/gately'
const DOCKER_CMD = `docker run -d --name gately --network host \\
  -v ./gately-data:/etc/gately \\
  -v /var/run/docker.sock:/var/run/docker.sock \\
  ghcr.io/wangwanjie/gately:latest
docker logs gately 2>&1 | grep -i "initial password"`

const promises = computed(() => [
  { icon: Scale, title: t('about.promises.openSource.title'), desc: t('about.promises.openSource.desc') },
  { icon: WifiOff, title: t('about.promises.noTelemetry.title'), desc: t('about.promises.noTelemetry.desc') },
  { icon: Eye, title: t('about.promises.visibleAddr.title'), desc: t('about.promises.visibleAddr.desc') },
  { icon: FileBraces, title: t('about.promises.plainConfig.title'), desc: t('about.promises.plainConfig.desc') },
])

// 与 README 的功能表保持一致
const features = computed(() => [
  { icon: Globe, title: t('about.features.ddns'), tags: [t('about.tags.aliyun'), 'Cloudflare', t('about.tags.tencent'), t('about.tags.huawei'), 'GoDaddy', 'Porkbun', 'dynv6', 'DuckDNS', t('about.tags.customCallback'), 'IPv4 / IPv6', 'Webhook'] },
  { icon: Radar, title: t('about.features.stun'), tags: ['TCP / UDP', t('about.tags.keepAlive'), 'UPnP / NAT-PMP', t('about.tags.builtinForward'), t('about.tags.addrChangeNotify'), t('about.tags.script')] },
  { icon: ArrowLeftRight, title: t('about.features.portforward'), tags: ['TCP / UDP', 'IPv4 / IPv6', t('about.tags.portRange'), t('about.tags.failover'), t('about.tags.connLimit'), t('about.tags.trafficStats')] },
  { icon: ShieldCheck, title: t('about.features.ssl'), tags: ['ACME DNS-01', t('about.tags.wildcard'), 'EAB', t('about.tags.autoRenew'), t('about.tags.uploadRead'), t('about.tags.mapToDir')] },
  { icon: Server, title: t('about.features.web'), tags: [t('about.tags.reverseProxy'), t('about.tags.redirect'), t('about.tags.staticFiles'), 'HTTPS', 'WebSocket', t('about.tags.proxyProtocol'), t('about.tags.oidcLogin'), t('about.tags.ipAcl')] },
  { icon: ShieldHalf, title: t('about.features.waf'), tags: ['Coraza', 'OWASP CRS v4', t('about.tags.rulesPerFile'), t('about.tags.customSecLang'), t('about.tags.detectOnly')] },
  { icon: Container, title: t('about.features.docker'), tags: [t('about.tags.containerOps'), t('about.tags.logs'), t('about.tags.imageCheck'), t('about.tags.oneClickUpgrade'), t('about.tags.autoRollback')] },
  { icon: SquareTerminal, title: t('about.features.terminal'), tags: ['SSH', t('about.tags.hostKeyTrust'), t('about.tags.localShell')] },
  { icon: Power, title: t('about.features.wol'), tags: [t('about.tags.magicPacket'), t('about.tags.onlineProbe')] },
  { icon: FolderSync, title: 'WebDAV', tags: [t('about.tags.multiUser'), t('about.tags.separateDir'), t('about.tags.readOnly')] },
  { icon: CalendarClock, title: t('about.features.cron'), tags: [t('about.tags.cronExpr'), t('about.tags.command'), t('about.tags.httpRequest'), t('about.tags.runHistory')] },
  { icon: LockKeyhole, title: t('about.features.security'), tags: [t('about.tags.twoFactor'), t('about.tags.loginLimit'), t('about.tags.secureEntry'), t('about.tags.lanOnly'), t('about.tags.apiToken'), 'DoH', t('about.tags.backupRestore')] },
  { icon: Import, title: t('about.features.luckyMigrate'), tags: ['.lkcf', t('about.tags.siteZip'), t('about.tags.portforward'), 'DDNS', 'STUN', t('about.tags.cert'), t('about.tags.reverseProxy'), 'WAF'] },
])

const stack = computed(() => ['Go', 'Vue 3', 'TypeScript', 'Element Plus', 'lego', 'Coraza', t('about.stackEmbed')])

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
    ElMessage.success(t('common.copied'))
  } catch {
    ElMessage.warning(t('about.copyFailed'))
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
