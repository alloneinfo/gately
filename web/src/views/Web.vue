<template>
  <div>
    <PageHeader :title="t('web.title')" :desc="t('web.desc')" :icon="Server">
      <el-button v-if="listeners.length" :icon="allListenersCollapsed ? ChevronsUpDown : ChevronsDownUp" @click="setAll(listeners.map((l) => l.key), !allListenersCollapsed)">
        {{ allListenersCollapsed ? t('common.expandAll') : t('common.collapseAll') }}
      </el-button>
      <CompactToggle v-model="compact" />
      <el-button :icon="SlidersHorizontal" @click="openSettings">{{ t('web.globalSettings') }}</el-button>
      <el-button type="primary" :icon="Plus" @click="openListener()">{{ t('web.addListener') }}</el-button>
    </PageHeader>

    <ModuleBanner v-if="moduleOff" :title="t('web.moduleOffTitle')" :desc="t('web.moduleOffDesc')" :loading="enabling" @enable="enableModule" />

    <div v-loading="loading" class="card-list">
      <el-empty v-if="!loading && !listeners.length" :description="t('web.empty')" />
      <EntityCard
        v-for="l in listeners"
        :key="l.key"
        :off="!l.enable || moduleOff"
        collapsible
        :collapsed="isCollapsed(l.key)"
        :collapsed-hint="t('web.sitesCollapsed', { count: siteRows(l).length })"
        @update:collapsed="(v: boolean) => setCollapsed(l.key, v)"
      >
        <template #lead>
          <el-switch :model-value="l.enable" @change="(v: string | number | boolean) => toggleListener(l, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ l.name || t('web.unnamedListener') }}</span>
          <span class="tags">
            <el-tag size="small" effect="plain" class="mono">{{ l.tls ? 'https' : 'http' }}://{{ l.listenIp || '*' }}:{{ l.port }}</el-tag>
            <el-tag size="small" type="info">{{ networkText(l.network) }}</el-tag>
            <el-tag v-if="l.acceptProxyProtocol" size="small" type="info">{{ t('web.acceptProxyTag') }}</el-tag>
          </span>
        </template>
        <template v-if="l.remark" #sub>{{ l.remark }}</template>
        <template #status>
          <StatusBadge v-bind="listenerStatus(l)" />
        </template>
        <template #extra>
          <div class="sites" :class="{ 'is-compact': compact }">
            <div v-for="row in siteRows(l)" :key="row.isDefault ? '__default' : row.index" class="site" :class="{ 'is-off': !row.site.enable }">
              <div class="site-switch">
                <el-switch :model-value="row.site.enable" size="small" @change="(v: string | number | boolean) => toggleSite(l, row, Boolean(v))" />
              </div>
              <div class="site-main">
                <div v-if="row.isDefault" class="site-domains"><el-tag size="small" type="warning">{{ t('web.defaultSite') }}</el-tag> <span class="meta">{{ t('web.unmatchedDomains') }}</span></div>
                <div v-else class="site-domains mono">{{ row.site.domains.join(', ') }}</div>
                <div v-if="row.site.name" class="meta">{{ row.site.name }}</div>
              </div>
              <div class="site-target">
                <el-tag size="small" effect="plain">{{ typeText(row.site.type) }}</el-tag>
                <span class="mono target">{{ summary(row.site) }}</span>
              </div>
              <div class="site-opts">
                <StatusBadge v-if="!row.site.enable" type="off" :text="t('web.statusOff')" />
                <el-tag v-if="row.site.webAuth" size="small" type="warning">{{ t('web.webAuthTag') }}</el-tag>
                <el-tag v-else-if="row.site.basicAuth" size="small" type="warning">{{ t('web.basicAuthTag') }}</el-tag>
                <el-tag v-else-if="row.site.forwardAuth?.url" size="small" type="warning">{{ t('web.forwardAuthTag') }}</el-tag>
                <el-tag v-if="row.site.locations?.length" size="small" type="info">{{ t('web.locationsTag', { count: row.site.locations.length }) }}</el-tag>
                <el-tag v-if="row.site.ipMode" size="small" type="info">{{ row.site.ipMode === 'whitelist' ? t('web.whitelist') : t('web.blacklist') }}</el-tag>
                <el-tag v-if="row.site.sendProxyProtocol" size="small" type="info">PROXY v2</el-tag>
                <el-tag v-if="downCount(l, row) > 0" size="small" type="danger">{{ t('web.upstreamsDown', { count: downCount(l, row) }) }}</el-tag>
                <el-tag v-if="row.site.insecureSkipVerify" size="small" type="info">{{ t('web.skipVerifyTag') }}</el-tag>
                <el-tag v-if="row.site.waf" size="small" :type="wafInstances.some((w) => w.key === row.site.waf) ? 'danger' : 'info'">
                  WAF{{ wafInstances.some((w) => w.key === row.site.waf) ? '' : t('web.wafMissingSuffix') }}
                </el-tag>
              </div>
              <div class="site-stats">
                <template v-if="row.site.key && l.stats?.[row.site.key]">
                  <span>{{ t('web.requestsCount', { count: l.stats[row.site.key].requests }) }}</span>
                  <span v-if="l.stats[row.site.key].errors" class="text-danger"> · {{ t('web.errorsCount', { count: l.stats[row.site.key].errors }) }}</span>
                </template>
                <span v-else class="muted">-</span>
              </div>
              <div class="site-actions">
                <el-button v-if="l.running" text size="small" :icon="BarChart3" @click="openStats(l, row)">{{ t('web.stats') }}</el-button>
                <el-button text size="small" :icon="Pencil" @click="openSite(l, row)">{{ t('common.edit') }}</el-button>
                <el-popconfirm v-if="row.isDefault" :title="t('web.resetDefaultConfirm')" width="260" @confirm="resetDefaultSite(l)">
                  <template #reference>
                    <el-button text size="small" type="danger" :icon="Trash2" :aria-label="t('common.reset')" />
                  </template>
                </el-popconfirm>
                <el-popconfirm v-else :title="t('web.deleteSiteConfirm')" @confirm="removeSite(l, row)">
                  <template #reference>
                    <el-button text size="small" type="danger" :icon="Trash2" :aria-label="t('web.deleteSite')" />
                  </template>
                </el-popconfirm>
              </div>
            </div>
            <div v-if="!siteRows(l).length" class="meta empty-sites">{{ t('web.noSites') }}</div>
          </div>
        </template>
        <template #actions>
          <el-button text size="small" type="primary" :icon="Plus" @click="openSite(l)">{{ t('web.addSite') }}</el-button>
          <el-button text size="small" :icon="Pencil" @click="openListener(l)">{{ t('web.editListener') }}</el-button>
          <el-popconfirm :title="t('web.deleteListenerConfirm')" width="240" @confirm="removeListener(l)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/webservice/logs" clearable />
    <SiteStats ref="statsDialog" />

    <el-dialog v-model="listenerVisible" :title="editingListener.key ? t('web.editListener') : t('web.addListener')" width="560px" :fullscreen="isMobile">
      <el-alert v-if="!editingListener.key" :title="t('web.newListenerHint')" type="info" :closable="false" show-icon class="listener-hint" />
      <el-form :model="editingListener" label-width="130px" :label-position="labelPosition">
        <el-form-item :label="t('common.name')"><el-input v-model="editingListener.name" /></el-form-item>
        <el-form-item :label="t('web.network')">
          <el-radio-group v-model="editingListener.network">
            <el-radio-button value="tcp">{{ t('web.networkDual') }}</el-radio-button>
            <el-radio-button value="tcp4">{{ t('web.networkIpv4') }}</el-radio-button>
            <el-radio-button value="tcp6">{{ t('web.networkIpv6') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('web.listenIp')"><el-input v-model="editingListener.listenIp" :placeholder="t('web.listenIpPlaceholder')" /></el-form-item>
        <el-form-item :label="t('web.port')" required><el-input-number v-model="editingListener.port" :min="1" :max="65535" /></el-form-item>
        <el-form-item label="HTTPS">
          <el-switch v-model="editingListener.tls" />
          <span class="help">{{ t('web.tlsHelp') }}</span>
        </el-form-item>
        <el-form-item v-if="editingListener.tls" :label="t('web.minTls')">
          <el-select v-model="editingListener.minTls" class="w-120">
            <el-option v-for="v in ['1.0', '1.1', '1.2', '1.3']" :key="v" :label="'TLS ' + v" :value="v" />
          </el-select>
        </el-form-item>
        <el-form-item v-if="editingListener.tls" label="HTTP/3">
          <el-switch v-model="editingListener.http3" />
          <span class="help">{{ t('web.http3Help') }}</span>
        </el-form-item>
        <el-form-item :label="t('web.acceptProxy')">
          <el-switch v-model="editingListener.acceptProxyProtocol" />
          <span class="help">{{ t('web.acceptProxyHelp') }}</span>
        </el-form-item>
        <el-form-item :label="t('common.remark')"><el-input v-model="editingListener.remark" /></el-form-item>
        <el-form-item :label="t('common.enable')"><el-switch v-model="editingListener.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="listenerVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="saveListener">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="siteVisible" :title="siteIsDefault ? t('web.defaultSite') : site.key ? t('web.editSite') : t('web.addSite')" width="760px" top="5vh" :fullscreen="isMobile">
      <el-form :model="site" label-width="130px" :label-position="labelPosition">
        <el-alert v-if="siteIsDefault" :title="t('web.defaultSiteHint')" type="info" :closable="false" show-icon class="listener-hint" />
        <el-tabs v-model="siteTab">
          <el-tab-pane :label="t('web.tabBasic')" name="basic">
            <el-form-item v-if="!siteIsDefault" :label="t('web.domains')" required>
              <el-input v-model="domainsText" type="textarea" :rows="2" class="mono" :placeholder="t('web.domainsPlaceholder')" />
            </el-form-item>
            <el-form-item :label="t('common.name')"><el-input v-model="site.name" /></el-form-item>
            <el-form-item :label="t('web.type')">
              <el-radio-group v-model="site.type">
                <el-radio-button value="proxy">{{ t('web.typeProxy') }}</el-radio-button>
                <el-radio-button value="redirect">{{ t('web.typeRedirect') }}</el-radio-button>
                <el-radio-button value="static">{{ t('web.typeStatic') }}</el-radio-button>
                <el-radio-button value="text">{{ t('web.typeText') }}</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <template v-if="site.type === 'proxy'">
              <el-form-item :label="t('web.targets')" required>
                <el-input v-model="targetsText" type="textarea" :rows="2" class="mono" :placeholder="t('web.targetsPlaceholder')" />
              </el-form-item>
              <el-form-item :label="t('web.options')">
                <el-checkbox v-model="site.insecureSkipVerify">{{ t('web.optSkipVerify') }}</el-checkbox>
                <el-checkbox v-model="site.useTargetHost">{{ t('web.optTargetHost') }}</el-checkbox>
                <el-checkbox v-model="site.disableKeepAlives">{{ t('web.optNoKeepAlive') }}</el-checkbox>
                <el-checkbox v-model="site.forwardedHeaders">{{ t('web.optForwarded') }}</el-checkbox>
                <el-checkbox v-model="site.sendProxyProtocol">{{ t('web.optSendProxy') }}</el-checkbox>
                <el-checkbox v-model="site.upstreamSniFromHost">{{ t('web.optSniFromHost') }}</el-checkbox>
                <div class="form-help">{{ t('web.optSniFromHostHelp') }}</div>
              </el-form-item>
              <el-form-item :label="t('web.locations')">
                <div class="locations">
                  <div v-for="(loc, i) in site.locations" :key="i" class="location-row">
                    <el-input v-model="loc.prefix" class="mono location-prefix" placeholder="/api/" />
                    <el-input
                      :model-value="(loc.targets ?? []).join(', ')"
                      class="mono location-targets"
                      :placeholder="t('web.locationTargetsPlaceholder')"
                      @update:model-value="(v: string) => (loc.targets = splitCsv(v))"
                    />
                    <el-input
                      :model-value="(loc.requestHeaders ?? []).join('\n')"
                      type="textarea"
                      :autosize="{ minRows: 1, maxRows: 4 }"
                      class="mono location-headers"
                      :placeholder="t('web.locationHeadersPlaceholder')"
                      @update:model-value="(v: string) => (loc.requestHeaders = lines(v))"
                    />
                    <el-button text type="danger" :icon="Trash2" @click="site.locations.splice(i, 1)" />
                  </div>
                  <el-button size="small" :icon="Plus" @click="site.locations.push({ prefix: '', targets: [], requestHeaders: [] })">{{ t('web.addLocation') }}</el-button>
                  <div class="form-help">{{ t('web.locationsHelp') }}</div>
                </div>
              </el-form-item>
            </template>
            <template v-if="site.type === 'redirect'">
              <el-form-item :label="t('web.redirectTo')" required>
                <el-input v-model="site.redirectTo" placeholder="https://example.com" />
                <div class="form-help">{{ t('web.redirectHostHelp') }}</div>
              </el-form-item>
              <el-form-item :label="t('web.statusCode')">
                <el-select v-model="site.redirectCode" class="w-120">
                  <el-option v-for="c in [301, 302, 307, 308]" :key="c" :label="c" :value="c" />
                </el-select>
                <el-checkbox v-model="site.keepPath" class="inline">{{ t('web.keepPath') }}</el-checkbox>
              </el-form-item>
            </template>
            <template v-if="site.type === 'static'">
              <el-form-item :label="t('web.root')" required><el-input v-model="site.root" placeholder="/data/www" /></el-form-item>
              <el-form-item :label="t('web.listDir')"><el-switch v-model="site.listDir" /></el-form-item>
              <el-form-item :label="t('web.cleanUrls')">
                <el-switch v-model="site.cleanUrls" />
                <div class="form-help">{{ t('web.cleanUrlsHelp') }}</div>
              </el-form-item>
              <el-form-item :label="t('web.notFoundPage')"><el-input v-model="site.notFoundPage" class="w-260" placeholder="404.html" /></el-form-item>
            </template>
            <template v-if="site.type === 'text'">
              <el-form-item :label="t('web.statusCode')"><el-input-number v-model="site.statusCode" :min="100" :max="599" /></el-form-item>
              <el-form-item label="Content-Type"><el-input v-model="site.contentType" placeholder="text/plain; charset=utf-8" /></el-form-item>
              <el-form-item :label="t('web.content')"><el-input v-model="site.text" type="textarea" :rows="4" /></el-form-item>
            </template>
            <el-form-item :label="t('web.accessLog')"><el-switch v-model="site.accessLog" /></el-form-item>
            <el-form-item :label="t('common.enable')"><el-switch v-model="site.enable" /></el-form-item>
          </el-tab-pane>
          <el-tab-pane :label="t('web.tabSecurity')" name="security">
            <el-form-item :label="t('web.auth')">
              <el-checkbox v-model="site.webAuth" :disabled="forwardAuthOn">{{ t('web.webAuth') }}</el-checkbox>
              <el-checkbox v-model="site.basicAuth" :disabled="forwardAuthOn">{{ t('web.basicAuth') }}</el-checkbox>
              <el-checkbox v-model="forwardAuthOn" :disabled="site.webAuth || site.basicAuth">{{ t('web.forwardAuth') }}</el-checkbox>
            </el-form-item>
            <template v-if="forwardAuthOn">
              <el-form-item :label="t('web.forwardAuthUrl')" required>
                <el-input v-model="site.forwardAuth.url" class="mono" placeholder="http://127.0.0.1:9091/api/authz/auth-request" />
                <div class="form-help">{{ t('web.forwardAuthUrlHelp') }}</div>
              </el-form-item>
              <el-form-item :label="t('web.forwardAuthLogin')">
                <el-input v-model="site.forwardAuth.loginUrl" class="mono" placeholder="https://auth.example.com/?rd={url}&rm={method}" />
                <div class="form-help">{{ t('web.forwardAuthLoginHelp') }}</div>
              </el-form-item>
              <el-form-item :label="t('web.forwardAuthHeaders')">
                <el-input v-model="fwdHeadersText" class="mono" placeholder="Remote-User, Remote-Groups, Remote-Email, Remote-Name" />
              </el-form-item>
            </template>
            <el-form-item v-if="site.webAuth || site.basicAuth" :label="t('web.users')">
              <el-input v-model="usersText" type="textarea" :rows="3" class="mono" :placeholder="t('web.usersPlaceholder')" />
            </el-form-item>
            <el-form-item v-if="site.webAuth" :label="t('web.thirdParty')">
              <el-input v-model="oauthUsersText" type="textarea" :rows="2" class="mono" :placeholder="t('web.oauthUsersPlaceholder')" />
              <div class="form-help">{{ t('web.providerHelp') }}</div>
            </el-form-item>
            <el-form-item :label="t('web.ipAccess')">
              <el-radio-group v-model="site.ipMode">
                <el-radio-button value="">{{ t('web.off') }}</el-radio-button>
                <el-radio-button value="whitelist">{{ t('web.whitelist') }}</el-radio-button>
                <el-radio-button value="blacklist">{{ t('web.blacklist') }}</el-radio-button>
              </el-radio-group>
            </el-form-item>
            <el-form-item v-if="site.ipMode" :label="t('web.ipRange')">
              <el-input v-model="ipText" type="textarea" :rows="3" class="mono" :placeholder="t('web.ipRangePlaceholder')" />
            </el-form-item>
            <SiteAccess v-model:site="site" :tls="siteTls" :port="sitePort" />
            <el-form-item :label="t('web.wafInstance')">
              <el-select v-model="site.waf" clearable :placeholder="t('web.wafNone')" class="w-260">
                <el-option v-for="w in wafInstances" :key="w.key" :label="w.name" :value="w.key" />
                <el-option v-if="site.waf && !wafInstances.some((w) => w.key === site.waf)" :label="site.waf + t('web.wafMissingSuffix')" :value="site.waf" />
              </el-select>
            </el-form-item>
            <el-form-item :label="t('web.cors')"><el-switch v-model="site.cors" /></el-form-item>
          </el-tab-pane>
          <el-tab-pane :label="t('web.tabTraffic')" name="traffic">
            <SiteTraffic v-model:site="site" />
          </el-tab-pane>
          <el-tab-pane :label="t('web.tabHeaders')" name="headers">
            <el-form-item :label="t('web.requestHeaders')">
              <el-input v-model="reqHeadersText" type="textarea" :rows="3" class="mono" :placeholder="t('web.requestHeadersPlaceholder')" />
            </el-form-item>
            <el-form-item :label="t('web.responseHeaders')">
              <el-input v-model="respHeadersText" type="textarea" :rows="3" class="mono" :placeholder="t('web.responseHeadersPlaceholder')" />
            </el-form-item>
          </el-tab-pane>
        </el-tabs>
      </el-form>
      <template #footer>
        <el-button @click="siteVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="saveSite">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>

    <el-dialog v-model="settingsVisible" :title="t('web.settingsTitle')" width="640px" :fullscreen="isMobile">
      <el-form v-if="settings" :model="settings" label-width="150px" :label-position="labelPosition">
        <el-form-item :label="t('web.enableModule')"><el-switch v-model="settings.enable" /></el-form-item>
        <el-form-item :label="t('web.trustedProxies')">
          <el-input v-model="trustedText" type="textarea" :rows="4" class="mono" :placeholder="t('web.trustedPlaceholder')" />
        </el-form-item>
        <el-divider content-position="left">{{ t('web.thirdPartyDivider') }}</el-divider>
        <el-form-item :label="t('web.callbackBase')">
          <el-input v-if="settings" v-model="settings.oauthCallbackBase" :placeholder="t('web.callbackPlaceholder')" />
          <div class="form-help">{{ t('web.callbackHelp', { url: (settings?.oauthCallbackBase || 'https://...').replace(/\/$/, '') + '/__gately_auth/oauth/callback' }) }}</div>
        </el-form-item>
        <OAuthProviders v-if="settings" v-model="settings.oauthProviders" />
        <AccessSettings v-if="settings" v-model:auto-ban-seconds="settings.autoBanSeconds" />
      </el-form>
      <template #footer>
        <el-button @click="settingsVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" @click="saveSettings">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { computed, onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { BarChart3, ChevronsDownUp, ChevronsUpDown, Pencil, Plus, Server, SlidersHorizontal, Trash2 } from 'lucide-vue-next'
import { api, type OAuthProvider } from '../api'
import PageHeader from '../components/PageHeader.vue'
import ModuleBanner from '../components/ModuleBanner.vue'
import CompactToggle from '../components/CompactToggle.vue'
import EntityCard from '../components/EntityCard.vue'
import { useCompactView } from '../composables/useCompactView'
import { useCollapsed } from '../composables/useCollapsed'
import OAuthProviders from '../components/OAuthProviders.vue'
import AccessSettings from '../components/AccessSettings.vue'
import SiteTraffic, { type Rewrite } from '../components/SiteTraffic.vue'
import SiteStats from '../components/SiteStats.vue'
import SiteAccess, { type PathRule, type ShareLink } from '../components/SiteAccess.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface Site {
  key?: string
  name: string
  enable: boolean
  domains: string[]
  type: string
  targets: string[] | null
  insecureSkipVerify: boolean
  useTargetHost: boolean
  disableKeepAlives: boolean
  sendProxyProtocol: boolean
  forwardedHeaders: boolean
  // 连接 HTTPS 后端时使用访客域名作为 SNI（后端按 SNI 分流时需要）
  upstreamSniFromHost: boolean
  locations: Location[]
  redirectTo: string
  redirectCode: number
  keepPath: boolean
  root: string
  listDir: boolean
  cleanUrls: boolean
  notFoundPage: string
  text: string
  contentType: string
  statusCode: number
  basicAuth: boolean
  webAuth: boolean
  users: string[] | null
  oauthUsers: string[] | null
  ipMode: string
  ipList: string[] | null
  requestHeaders: string[] | null
  responseHeaders: string[] | null
  cors: boolean
  accessLog: boolean
  waf: string
  remark: string
  authOptional: boolean
  forwardAuth: { url: string; loginUrl: string; copyHeaders: string[] | null }
  countryMode: string
  countries: string[] | null
  rules: PathRule[]
  shareLinks: ShareLink[]
  lbPolicy: string
  health: { enable: boolean; mode: string; path: string; interval: number; timeout: number; fails: number; passes: number }
  compress: boolean
  rateLimit: { enable: boolean; rate: number; burst: number; maxConns: number }
  cache: { enable: boolean; ttl: number; maxMb: number }
  rewrites: Rewrite[]
}

// Location 把某个路径前缀分到另一组后端
interface Location {
  prefix: string
  targets: string[] | null
  requestHeaders: string[] | null
}

interface Listener {
  key?: string
  name: string
  enable: boolean
  network: string
  listenIp: string
  port: number
  tls: boolean
  minTls: string
  http3: boolean
  acceptProxyProtocol: boolean
  sites: Site[] | null
  default: Site
  remark: string
}

interface ListenerView extends Listener {
  key: string
  running: boolean
  error?: string
  stats: Record<string, { requests: number; active: number; errors: number }>
  upstreams?: Record<string, { url: string; healthy: boolean }[]>
}

interface SiteRow {
  site: Site
  index: number
  isDefault: boolean
}

const { t } = useI18n()
const compact = useCompactView('web')
const listeners = ref<ListenerView[]>([])
// 每个监听下的站点列表可单独收起，状态按监听 key 持久化
const { isCollapsed, setCollapsed, setAll, allCollapsed } = useCollapsed('web')
const allListenersCollapsed = allCollapsed(() => listeners.value.map((l) => l.key))
const wafInstances = ref<{ key: string; name: string }[]>([])
const loading = ref(true)
const saving = ref(false)
const listenerVisible = ref(false)
const editingListener = ref<Listener>(emptyListener())
const siteVisible = ref(false)
const siteTab = ref('basic')
const site = ref<Site>(emptySite())
const siteIsDefault = ref(false)
let siteOwner: ListenerView | null = null
// 分享链接地址需要所属监听器的协议与端口
const siteTls = ref(false)
const sitePort = ref(0)
let siteIndex = -1
const settingsVisible = ref(false)
const { isMobile, labelPosition } = useFormLayout()
// moduleEnabled 为模块全局开关，undefined 表示尚未加载
const moduleEnabled = ref<boolean>()
const moduleOff = computed(() => moduleEnabled.value === false)
const enabling = ref(false)
const settings = ref<{ enable: boolean; trustedProxies: string[] | null; oauthCallbackBase: string; oauthProviders: OAuthProvider[]; autoBanSeconds: number }>()
let timer: number | undefined

const lines = (v: string) => v.split('\n').map((s) => s.trim()).filter(Boolean)
const bind = (get: () => string[] | null | undefined, set: (v: string[]) => void) =>
  computed({ get: () => (get() ?? []).join('\n'), set: (v: string) => set(lines(v)) })
const domainsText = bind(() => site.value.domains, (v) => (site.value.domains = v))
const targetsText = bind(() => site.value.targets, (v) => (site.value.targets = v))
const usersText = bind(() => site.value.users, (v) => (site.value.users = v))
const oauthUsersText = bind(() => site.value.oauthUsers, (v) => (site.value.oauthUsers = v))
const ipText = bind(() => site.value.ipList, (v) => (site.value.ipList = v))
const reqHeadersText = bind(() => site.value.requestHeaders, (v) => (site.value.requestHeaders = v))
const respHeadersText = bind(() => site.value.responseHeaders, (v) => (site.value.responseHeaders = v))
const fwdHeadersText = computed({
  get: () => (site.value.forwardAuth?.copyHeaders ?? []).join(', '),
  set: (v: string) => (site.value.forwardAuth.copyHeaders = splitCsv(v)),
})
// 转发认证以地址非空表示启用，勾选框单独记录，取消勾选时保存为空地址
const forwardAuthOn = ref(false)
const splitCsv = (v: string) =>
  v
    .split(/[\s,，]+/)
    .map((x) => x.trim())
    .filter(Boolean)
const trustedText = bind(() => settings.value?.trustedProxies, (v) => settings.value && (settings.value.trustedProxies = v))

function emptySite(): Site {
  return {
    name: '',
    enable: true,
    domains: [],
    type: 'proxy',
    targets: [],
    insecureSkipVerify: false,
    useTargetHost: false,
    disableKeepAlives: false,
    sendProxyProtocol: false,
    forwardedHeaders: true,
    upstreamSniFromHost: false,
    locations: [],
    redirectTo: '',
    redirectCode: 302,
    keepPath: true,
    root: '',
    listDir: false,
    cleanUrls: false,
    notFoundPage: '',
    text: '',
    contentType: '',
    statusCode: 200,
    basicAuth: false,
    webAuth: false,
    users: [],
    oauthUsers: [],
    ipMode: '',
    ipList: [],
    requestHeaders: [],
    responseHeaders: [],
    cors: false,
    accessLog: false,
    waf: '',
    remark: '',
    authOptional: false,
    forwardAuth: { url: '', loginUrl: '', copyHeaders: [] },
    countryMode: '',
    countries: [],
    rules: [],
    shareLinks: [],
    lbPolicy: '',
    health: { enable: false, mode: 'http', path: '/', interval: 10, timeout: 3, fails: 3, passes: 2 },
    compress: false,
    rateLimit: { enable: false, rate: 10, burst: 20, maxConns: 0 },
    cache: { enable: false, ttl: 0, maxMb: 64 },
    rewrites: [],
  }
}

function emptyListener(): Listener {
  return {
    name: '',
    enable: true,
    network: 'tcp',
    listenIp: '',
    port: 443,
    tls: true,
    minTls: '1.2',
    http3: false,
    acceptProxyProtocol: false,
    sites: [],
    default: { ...emptySite(), enable: false },
    remark: '',
  }
}

function networkText(n: string) {
  const map: Record<string, string> = { tcp: t('web.networkDual'), tcp4: t('web.networkIpv4'), tcp6: t('web.networkIpv6') }
  return map[n] ?? n
}

// listenerStatus 计算监听展示状态：停用的监听只显示“已停用”
function listenerStatus(l: ListenerView): { type: StatusType; text: string; title?: string } {
  if (moduleOff.value) {
    return { type: 'off', text: t('web.moduleOff') }
  }
  if (!l.enable) {
    return { type: 'off', text: t('web.statusOff') }
  }
  if (l.error) {
    return { type: 'error', text: l.error, title: l.error }
  }
  if (l.running) {
    return { type: 'running', text: t('common.running') }
  }
  return { type: 'warn', text: t('web.statusNotRunning') }
}

function typeText(ty: string) {
  const map: Record<string, string> = { proxy: t('web.typeProxy'), redirect: t('web.typeRedirect'), static: t('web.typeStatic'), text: t('web.typeText') }
  return map[ty] ?? ty
}

function summary(s: Site) {
  switch (s.type) {
    case 'proxy':
      return (s.targets ?? []).join(' | ')
    case 'redirect':
      return `${s.redirectCode} → ${s.redirectTo}`
    case 'static':
      return s.root
    default:
      return `${s.statusCode} ${s.text.slice(0, 30)}`
  }
}

function siteRows(l: ListenerView): SiteRow[] {
  const rows = (l.sites ?? []).map((s, i) => ({ site: s, index: i, isDefault: false }))
  if (l.default.enable || (l.default.targets ?? []).length || l.default.type !== 'proxy') {
    rows.push({ site: l.default, index: -1, isDefault: true })
  }
  return rows
}

// 去掉运行状态字段，得到可提交的监听配置
function plain(l: ListenerView): Listener {
  // 剥离仅用于展示的运行态字段，后端保存接口会拒绝未知字段
  const { running: _r, error: _e, stats: _s, upstreams: _u, ...rest } = l
  return JSON.parse(JSON.stringify(rest))
}

async function putListener(l: Listener) {
  const { key, ...body } = l
  await api.put(`/api/webservice/listeners/${key}`, body)
}

async function load() {
  try {
    listeners.value = await api.get<ListenerView[]>('/api/webservice/listeners')
  } finally {
    loading.value = false
  }
}

function openListener(l?: ListenerView) {
  editingListener.value = l ? plain(l) : emptyListener()
  listenerVisible.value = true
}

async function saveListener() {
  saving.value = true
  try {
    const { key, ...body } = editingListener.value
    if (key) {
      await api.put(`/api/webservice/listeners/${key}`, body)
    } else {
      await api.post('/api/webservice/listeners', body)
    }
    listenerVisible.value = false
    ElMessage.success(t('common.saved'))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggleListener(l: ListenerView, enable: boolean) {
  try {
    await api.put(`/api/webservice/listeners/${l.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeListener(l: ListenerView) {
  try {
    await api.del(`/api/webservice/listeners/${l.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

const statsDialog = ref<InstanceType<typeof SiteStats>>()

function openStats(l: ListenerView, row: SiteRow) {
  statsDialog.value?.open(l.key ?? '', row.isDefault ? 'default' : (row.site.key ?? ''), row.site.name || row.site.domains?.[0] || t('web.defaultSite'))
}

// downCount 统计站点中不可用的后端数量
function downCount(l: ListenerView, row: SiteRow) {
  const k = row.isDefault ? 'default' : (row.site.key ?? '')
  return (l.upstreams?.[k] ?? []).filter((u) => !u.healthy).length
}

function openSite(l: ListenerView, row?: SiteRow) {
  siteOwner = l
  siteTls.value = l.tls
  sitePort.value = l.port
  siteIndex = row ? row.index : -1
  siteIsDefault.value = !!row?.isDefault
  // 旧配置中新增字段为 null，合并默认值后保证数组可直接编辑
  const draft: Site = row ? JSON.parse(JSON.stringify(row.site)) : emptySite()
  draft.rules = draft.rules ?? []
  draft.shareLinks = draft.shareLinks ?? []
  draft.countries = draft.countries ?? []
  draft.rewrites = draft.rewrites ?? []
  draft.locations = draft.locations ?? []
  draft.forwardAuth = Object.assign({ url: '', loginUrl: '', copyHeaders: [] }, draft.forwardAuth)
  forwardAuthOn.value = !!draft.forwardAuth.url
  // 旧配置没有这些对象，补全默认值后可直接编辑
  const defaults = emptySite()
  draft.health = { ...defaults.health, ...(draft.health ?? {}) }
  draft.rateLimit = { ...defaults.rateLimit, ...(draft.rateLimit ?? {}) }
  draft.cache = { ...defaults.cache, ...(draft.cache ?? {}) }
  site.value = draft
  siteTab.value = 'basic'
  siteVisible.value = true
}

async function saveSite() {
  if (!siteOwner) {
    return
  }
  if (!forwardAuthOn.value) {
    site.value.forwardAuth = { url: '', loginUrl: '', copyHeaders: [] }
  }
  const l = plain(siteOwner)
  l.sites = l.sites ?? []
  if (siteIsDefault.value) {
    l.default = site.value
  } else if (siteIndex >= 0) {
    l.sites[siteIndex] = site.value
  } else {
    l.sites.push(site.value)
  }
  saving.value = true
  try {
    await putListener(l)
    siteVisible.value = false
    ElMessage.success(t('web.savedReloaded'))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggleSite(lv: ListenerView, row: SiteRow, enable: boolean) {
  const l = plain(lv)
  if (row.isDefault) {
    l.default.enable = enable
  } else if (l.sites) {
    l.sites[row.index].enable = enable
  }
  try {
    await putListener(l)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function removeSite(lv: ListenerView, row: SiteRow) {
  const l = plain(lv)
  l.sites?.splice(row.index, 1)
  try {
    await putListener(l)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

// resetDefaultSite 清空默认站点配置并保持停用（默认站点无法删除）
async function resetDefaultSite(lv: ListenerView) {
  const l = plain(lv)
  l.default = { ...emptySite(), key: lv.default.key, enable: false }
  try {
    await putListener(l)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function fetchSettings() {
  const s = await api.get<{ enable: boolean; trustedProxies: string[] | null; oauthCallbackBase: string; oauthProviders: OAuthProvider[] | null; autoBanSeconds: number }>('/api/webservice/settings')
  moduleEnabled.value = s.enable
  return { ...s, oauthProviders: s.oauthProviders ?? [] }
}

async function openSettings() {
  settings.value = await fetchSettings()
  settingsVisible.value = true
}

// enableModule 在横幅中一键启用模块
async function enableModule() {
  enabling.value = true
  try {
    const s = await fetchSettings()
    await api.put('/api/webservice/settings', { ...s, enable: true })
    ElMessage.success(t('web.moduleEnabled'))
    await Promise.all([fetchSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    enabling.value = false
  }
}

async function saveSettings() {
  try {
    await api.put('/api/webservice/settings', settings.value)
    settingsVisible.value = false
    ElMessage.success(t('common.saved'))
    await Promise.all([fetchSettings(), load()])
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  fetchSettings()
  wafInstances.value = await api.get('/api/waf/instances')
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.listener-hint {
  margin-bottom: 16px;
}
.locations {
  display: flex;
  flex-direction: column;
  gap: 8px;
  width: 100%;
}
.location-row {
  display: flex;
  gap: 6px;
  align-items: flex-start;
}
.location-prefix {
  width: 120px;
  flex: none;
}
.location-targets,
.location-headers {
  flex: 1;
  min-width: 0;
}
.sites {
  border: 1px solid var(--g-border);
  border-radius: 8px;
  overflow: hidden;
}
.site {
  display: grid;
  grid-template-columns: 40px minmax(0, 1.5fr) minmax(0, 1.8fr) minmax(0, 1.3fr) 110px auto;
  gap: 8px 14px;
  align-items: center;
  padding: 9px 8px 9px 12px;
  font-size: 13px;
}
.site + .site {
  border-top: 1px solid var(--g-border);
}
.site.is-off .site-main,
.site.is-off .site-target,
.site.is-off .site-stats {
  opacity: 0.55;
}
.site-main,
.site-target {
  min-width: 0;
}
.site-domains {
  word-break: break-all;
}
.target {
  margin-left: 6px;
  font-size: 12px;
  word-break: break-all;
}
.site-opts {
  display: flex;
  flex-wrap: wrap;
  gap: 4px;
}
.site-stats {
  font-size: 12px;
  white-space: nowrap;
}
.site-actions {
  display: flex;
  justify-content: flex-end;
}
.site-actions :deep(.el-button + .el-button) {
  margin-left: 0;
}
.empty-sites {
  padding: 10px 12px;
}
.inline {
  margin-left: 16px;
}
.w-120 {
  width: 120px;
}
.w-260 {
  width: 260px;
}
.el-checkbox {
  white-space: normal;
  height: auto;
  min-height: 32px;
}
/* 紧凑视图：站点行压成单行，域名与说明同行，超长内容省略（仅桌面宽度） */
@media (min-width: 768px) {
  .sites.is-compact .site {
    gap: 4px 12px;
    padding: 3px 8px 3px 12px;
    font-size: 12px;
  }
  .sites.is-compact .site-main {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .sites.is-compact .site-domains,
  .sites.is-compact .site-main .meta,
  .sites.is-compact .site-target {
    min-width: 0;
    overflow: hidden;
    text-overflow: ellipsis;
    white-space: nowrap;
    word-break: normal;
  }
  /* 域名优先完整显示，名称说明只占剩余空间 */
  .sites.is-compact .site {
    grid-template-columns: 40px minmax(0, 1.9fr) minmax(0, 1.6fr) minmax(0, 1fr) 100px auto;
  }
  .sites.is-compact .site-main .site-domains {
    flex: 0 1 auto;
  }
  .sites.is-compact .site-main .meta {
    flex: 1 1 0;
  }
  .sites.is-compact .site-opts {
    flex-wrap: nowrap;
    overflow: hidden;
  }
  .sites.is-compact .site-opts :deep(.el-tag) {
    flex-shrink: 0;
  }
}
@media (max-width: 767.98px) {
  .site {
    grid-template-columns: auto minmax(0, 1fr) auto;
    padding: 10px;
  }
  .site-switch {
    grid-row: 1;
    grid-column: 1;
  }
  .site-main {
    grid-row: 1;
    grid-column: 2;
  }
  .site-actions {
    grid-row: 1;
    grid-column: 3;
  }
  .site-target,
  .site-opts,
  .site-stats {
    grid-column: 2 / 4;
  }
  .site-opts:empty {
    display: none;
  }
  .w-260 {
    width: 100%;
  }
  .inline {
    margin-left: 0;
  }
}
</style>
