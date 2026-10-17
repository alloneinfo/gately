<template>
  <div>
    <PageHeader :title="t('waf.title')" :desc="t('waf.desc')" :icon="ShieldHalf">
      <CompactToggle v-model="compact" />
      <el-button type="primary" :icon="Plus" @click="openEditor()">{{ t('waf.addInstance') }}</el-button>
    </PageHeader>

    <div class="card-list" :class="{ 'is-compact': compact }">
      <el-empty v-if="!instances.length" :description="t('waf.empty')" />
      <EntityCard :compact="compact" v-for="w in instances" :key="w.key" :off="!w.enable">
        <template #lead>
          <el-switch :model-value="w.enable" @change="(v: string | number | boolean) => toggle(w, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ w.name }}</span>
          <el-tag :type="w.mode === 'block' ? 'danger' : 'warning'" size="small" effect="plain">{{ w.mode === 'block' ? t('waf.modeBlockShort') : t('waf.modeDetectShort') }}</el-tag>
        </template>
        <template #sub><span class="mono">{{ w.key }}</span><span v-if="w.remark"> · {{ w.remark }}</span></template>
        <template #status>
          <StatusBadge v-bind="instStatus(w)" />
        </template>
        <Field :label="t('waf.ruleSets')" wide>
          <div v-for="(r, i) in w.ruleSets ?? []" :key="i" class="ruleset-line" :class="{ off: !r.enable }">
            <span class="dot" />
            {{ r.type === 'crs' ? 'OWASP CRS' : t('waf.custom') }}{{ r.remark ? t('waf.remarkSuffix', { remark: r.remark }) : '' }}
            <span v-if="r.type === 'crs'" class="meta">{{ r.param ? t('waf.ruleFiles', { count: r.param.split('\n').filter(Boolean).length }) : t('waf.allRules') }}</span>
          </div>
          <span v-if="!(w.ruleSets ?? []).length" class="muted">{{ t('common.none') }}</span>
        </Field>
        <Field :label="t('waf.thresholdParanoia')">{{ t('waf.thresholdValue', { inbound: w.inboundThreshold, outbound: w.outboundThreshold, level: w.paranoia }) }}</Field>
        <Field :label="t('waf.blocked')"><span :class="{ 'text-danger': w.blocked > 0 }">{{ w.blocked }}</span></Field>
        <Field :label="t('waf.matched')">{{ w.matched }}</Field>
        <template v-if="w.error && w.enable" #extra>
          <div class="err-line">{{ w.error }}</div>
        </template>
        <template #actions>
          <el-button text size="small" :icon="Pencil" @click="openEditor(w)">{{ t('common.edit') }}</el-button>
          <el-popconfirm :title="t('waf.deleteConfirm')" width="240" @confirm="remove(w)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">{{ t('common.delete') }}</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/waf/logs" :title="t('waf.logTitle')" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? t('waf.editInstance') : t('waf.addInstance')" width="760px" top="5vh" :fullscreen="isMobile">
      <el-form :model="editing" label-width="120px" :label-position="labelPosition">
        <el-form-item :label="t('common.name')"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item :label="t('waf.mode')">
          <el-radio-group v-model="editing.mode">
            <el-radio-button value="block">{{ t('waf.modeBlock') }}</el-radio-button>
            <el-radio-button value="detect">{{ t('waf.modeDetect') }}</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item :label="t('waf.paranoia')">
          <el-slider v-model="editing.paranoia" :min="1" :max="4" show-stops class="slider" />
          <span class="help">{{ t('waf.paranoiaHelp') }}</span>
        </el-form-item>
        <el-form-item :label="t('waf.inbound')"><el-input-number v-model="editing.inboundThreshold" :min="1" :max="100" /></el-form-item>
        <el-form-item :label="t('waf.outbound')"><el-input-number v-model="editing.outboundThreshold" :min="1" :max="100" /></el-form-item>
        <el-divider content-position="left">{{ t('waf.ruleSets') }}</el-divider>
        <div v-for="(r, i) in editing.ruleSets" :key="i" class="ruleset">
          <div class="ruleset-head">
            <el-switch v-model="r.enable" size="small" />
            <el-radio-group v-model="r.type" size="small">
              <el-radio-button value="crs">OWASP CRS</el-radio-button>
              <el-radio-button value="custom">{{ t('waf.customSecLang') }}</el-radio-button>
            </el-radio-group>
            <el-input v-model="r.remark" size="small" :placeholder="t('common.remark')" class="rs-remark" />
            <el-button text type="danger" size="small" :icon="Trash2" :aria-label="t('waf.deleteRuleSet')" @click="editing.ruleSets.splice(i, 1)" />
          </div>
          <el-select
            v-if="r.type === 'crs'"
            :model-value="crsSelected(r)"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            :placeholder="t('waf.crsPlaceholder')"
            style="width: 100%"
            @update:model-value="(v: string[]) => (r.param = v.join('\n'))"
          >
            <el-option v-for="f in crsFiles" :key="f" :label="f" :value="f" />
          </el-select>
          <el-input v-else v-model="r.param" type="textarea" :rows="4" class="mono" placeholder='SecRule REQUEST_URI "@beginsWith /admin" "id:10001,phase:1,deny,status:403"' />
        </div>
        <el-button size="small" :icon="Plus" @click="editing.ruleSets.push({ enable: true, remark: '', type: 'crs', param: '' })">{{ t('waf.addRuleSet') }}</el-button>
        <el-form-item :label="t('common.remark')" class="remark"><el-input v-model="editing.remark" /></el-form-item>
        <el-form-item :label="t('common.enable')"><el-switch v-model="editing.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">{{ t('common.cancel') }}</el-button>
        <el-button type="primary" :loading="saving" @click="save">{{ t('common.save') }}</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { useI18n } from 'vue-i18n'
import { ElMessage } from 'element-plus'
import { Pencil, Plus, ShieldHalf, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import CompactToggle from '../components/CompactToggle.vue'
import EntityCard from '../components/EntityCard.vue'
import { useCompactView } from '../composables/useCompactView'
import Field from '../components/Field.vue'
import StatusBadge, { type StatusType } from '../components/StatusBadge.vue'
import LogCard from '../components/LogCard.vue'
import { useFormLayout } from '../composables/useIsMobile'

interface RuleSet {
  enable: boolean
  remark: string
  type: string
  param: string
}

interface Instance {
  key?: string
  name: string
  enable: boolean
  mode: string
  paranoia: number
  inboundThreshold: number
  outboundThreshold: number
  ruleSets: RuleSet[]
  remark: string
}

interface InstanceView extends Instance {
  key: string
  error?: string
  blocked: number
  matched: number
}

const { t } = useI18n()
const compact = useCompactView('waf')
const { isMobile, labelPosition } = useFormLayout()
const instances = ref<InstanceView[]>([])
const crsFiles = ref<string[]>([])
const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Instance>(empty())
let timer: number | undefined

function empty(): Instance {
  return { name: '', enable: true, mode: 'block', paranoia: 1, inboundThreshold: 5, outboundThreshold: 4, ruleSets: [{ enable: true, remark: t('waf.defaultRuleSetRemark'), type: 'crs', param: '' }], remark: '' }
}

// instStatus 计算实例展示状态：停用的实例只显示“已停用”
function instStatus(w: InstanceView): { type: StatusType; text: string; title?: string } {
  if (!w.enable) {
    return { type: 'off', text: t('waf.statusOff') }
  }
  if (w.error) {
    return { type: 'error', text: t('waf.statusCompileFailed'), title: w.error }
  }
  return { type: 'running', text: w.mode === 'block' ? t('waf.statusProtecting') : t('waf.statusDetecting') }
}

function crsSelected(r: RuleSet) {
  return r.param.split('\n').map((s) => s.trim()).filter(Boolean)
}

async function load() {
  instances.value = await api.get('/api/waf/instances')
}

function openEditor(row?: InstanceView) {
  if (row) {
    const { error: _e, blocked: _b, matched: _m, ...inst } = row
    editing.value = JSON.parse(JSON.stringify(inst))
    editing.value.ruleSets = editing.value.ruleSets ?? []
  } else {
    editing.value = empty()
  }
  editorVisible.value = true
}

async function save() {
  saving.value = true
  try {
    const { key, ...body } = editing.value
    if (key) {
      await api.put(`/api/waf/instances/${key}`, body)
    } else {
      await api.post('/api/waf/instances', body)
    }
    editorVisible.value = false
    ElMessage.success(t('waf.savedApplied'))
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

async function toggle(row: InstanceView, enable: boolean) {
  try {
    await api.put(`/api/waf/instances/${row.key}/enable`, { enable })
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

async function remove(row: InstanceView) {
  try {
    await api.del(`/api/waf/instances/${row.key}`)
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  }
}

onMounted(async () => {
  crsFiles.value = await api.get('/api/waf/crs')
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.ruleset-line {
  display: flex;
  align-items: center;
  gap: 6px;
  flex-wrap: wrap;
}
.ruleset-line .dot {
  width: 6px;
  height: 6px;
  border-radius: 50%;
  background: var(--el-color-success);
}
.ruleset-line.off {
  color: var(--el-text-color-secondary);
}
.ruleset-line.off .dot {
  background: var(--el-text-color-placeholder);
}
.err-line {
  font-size: 12px;
  color: var(--el-color-danger);
  word-break: break-all;
}
.slider {
  width: 240px;
  margin-right: 8px;
}
.ruleset {
  border: 1px solid var(--g-border);
  border-radius: 8px;
  padding: 10px;
  margin-bottom: 10px;
}
.ruleset-head {
  display: flex;
  flex-wrap: wrap;
  gap: 8px 10px;
  align-items: center;
  margin-bottom: 8px;
}
.rs-remark {
  width: 200px;
}
.remark {
  margin-top: 16px;
}
@media (max-width: 767.98px) {
  .slider {
    width: 100%;
    margin: 0 8px;
  }
  .rs-remark {
    width: auto;
    flex: 1;
    min-width: 120px;
  }
}
</style>
