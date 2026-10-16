<template>
  <div>
    <PageHeader title="WAF 防火墙" desc="基于 Coraza 与 OWASP 核心规则集（CRS），在 Web 服务的站点中引用实例即可启用" :icon="ShieldHalf">
      <el-button type="primary" :icon="Plus" @click="openEditor()">添加实例</el-button>
    </PageHeader>

    <div class="card-list">
      <el-empty v-if="!instances.length" description="暂无实例" />
      <EntityCard v-for="w in instances" :key="w.key" :off="!w.enable">
        <template #lead>
          <el-switch :model-value="w.enable" @change="(v: string | number | boolean) => toggle(w, Boolean(v))" />
        </template>
        <template #title>
          <span>{{ w.name }}</span>
          <el-tag :type="w.mode === 'block' ? 'danger' : 'warning'" size="small" effect="plain">{{ w.mode === 'block' ? '拦截' : '仅检测' }}</el-tag>
        </template>
        <template #sub><span class="mono">{{ w.key }}</span><span v-if="w.remark"> · {{ w.remark }}</span></template>
        <template #status>
          <StatusBadge v-bind="instStatus(w)" />
        </template>
        <Field label="规则组" wide>
          <div v-for="(r, i) in w.ruleSets ?? []" :key="i" class="ruleset-line" :class="{ off: !r.enable }">
            <span class="dot" />
            {{ r.type === 'crs' ? 'OWASP CRS' : '自定义' }}{{ r.remark ? '：' + r.remark : '' }}
            <span v-if="r.type === 'crs'" class="meta">（{{ r.param ? r.param.split('\n').filter(Boolean).length + ' 个文件' : '全部规则' }}）</span>
          </div>
          <span v-if="!(w.ruleSets ?? []).length" class="muted">无</span>
        </Field>
        <Field label="阈值 / 偏执等级">入站 {{ w.inboundThreshold }} · 出站 {{ w.outboundThreshold }} · PL{{ w.paranoia }}</Field>
        <Field label="已拦截"><span :class="{ 'text-danger': w.blocked > 0 }">{{ w.blocked }}</span></Field>
        <Field label="规则命中">{{ w.matched }}</Field>
        <template v-if="w.error && w.enable" #extra>
          <div class="err-line">{{ w.error }}</div>
        </template>
        <template #actions>
          <el-button text size="small" :icon="Pencil" @click="openEditor(w)">编辑</el-button>
          <el-popconfirm title="删除该实例？引用它的站点将不再检查" width="240" @confirm="remove(w)">
            <template #reference>
              <el-button text size="small" type="danger" :icon="Trash2">删除</el-button>
            </template>
          </el-popconfirm>
        </template>
      </EntityCard>
    </div>

    <LogCard url="/api/waf/logs" title="拦截与命中日志" clearable />

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑实例' : '添加实例'" width="760px" top="5vh" :fullscreen="isMobile">
      <el-form :model="editing" label-width="120px" :label-position="labelPosition">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="模式">
          <el-radio-group v-model="editing.mode">
            <el-radio-button value="block">拦截</el-radio-button>
            <el-radio-button value="detect">仅检测（只记录日志）</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="偏执等级">
          <el-slider v-model="editing.paranoia" :min="1" :max="4" show-stops class="slider" />
          <span class="help">越高越严格，误报也越多，建议从 1 开始</span>
        </el-form-item>
        <el-form-item label="入站阈值"><el-input-number v-model="editing.inboundThreshold" :min="1" :max="100" /></el-form-item>
        <el-form-item label="出站阈值"><el-input-number v-model="editing.outboundThreshold" :min="1" :max="100" /></el-form-item>
        <el-divider content-position="left">规则组</el-divider>
        <div v-for="(r, i) in editing.ruleSets" :key="i" class="ruleset">
          <div class="ruleset-head">
            <el-switch v-model="r.enable" size="small" />
            <el-radio-group v-model="r.type" size="small">
              <el-radio-button value="crs">OWASP CRS</el-radio-button>
              <el-radio-button value="custom">自定义 SecLang</el-radio-button>
            </el-radio-group>
            <el-input v-model="r.remark" size="small" placeholder="备注" class="rs-remark" />
            <el-button text type="danger" size="small" :icon="Trash2" aria-label="删除规则组" @click="editing.ruleSets.splice(i, 1)" />
          </div>
          <el-select
            v-if="r.type === 'crs'"
            :model-value="crsSelected(r)"
            multiple
            filterable
            collapse-tags
            collapse-tags-tooltip
            placeholder="留空表示启用全部规则文件"
            style="width: 100%"
            @update:model-value="(v: string[]) => (r.param = v.join('\n'))"
          >
            <el-option v-for="f in crsFiles" :key="f" :label="f" :value="f" />
          </el-select>
          <el-input v-else v-model="r.param" type="textarea" :rows="4" class="mono" placeholder='SecRule REQUEST_URI "@beginsWith /admin" "id:10001,phase:1,deny,status:403"' />
        </div>
        <el-button size="small" :icon="Plus" @click="editing.ruleSets.push({ enable: true, remark: '', type: 'crs', param: '' })">添加规则组</el-button>
        <el-form-item label="备注" class="remark"><el-input v-model="editing.remark" /></el-form-item>
        <el-form-item label="启用"><el-switch v-model="editing.enable" /></el-form-item>
      </el-form>
      <template #footer>
        <el-button @click="editorVisible = false">取消</el-button>
        <el-button type="primary" :loading="saving" @click="save">保存</el-button>
      </template>
    </el-dialog>
  </div>
</template>

<script setup lang="ts">
import { onBeforeUnmount, onMounted, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Pencil, Plus, ShieldHalf, Trash2 } from 'lucide-vue-next'
import { api } from '../api'
import PageHeader from '../components/PageHeader.vue'
import EntityCard from '../components/EntityCard.vue'
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

const { isMobile, labelPosition } = useFormLayout()
const instances = ref<InstanceView[]>([])
const crsFiles = ref<string[]>([])
const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Instance>(empty())
let timer: number | undefined

function empty(): Instance {
  return { name: '', enable: true, mode: 'block', paranoia: 1, inboundThreshold: 5, outboundThreshold: 4, ruleSets: [{ enable: true, remark: 'OWASP 核心规则', type: 'crs', param: '' }], remark: '' }
}

// instStatus 计算实例展示状态：停用的实例只显示“已停用”
function instStatus(w: InstanceView): { type: StatusType; text: string; title?: string } {
  if (!w.enable) {
    return { type: 'off', text: '已停用' }
  }
  if (w.error) {
    return { type: 'error', text: '编译失败', title: w.error }
  }
  return { type: 'running', text: w.mode === 'block' ? '防护中' : '检测中' }
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
    ElMessage.success('已保存并生效')
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
