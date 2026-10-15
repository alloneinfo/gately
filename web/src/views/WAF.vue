<template>
  <div>
    <div class="page-header">
      <h2>WAF 防火墙</h2>
      <el-button type="primary" @click="openEditor()">添加实例</el-button>
    </div>
    <el-alert type="info" :closable="false" class="intro">
      基于 Coraza 与 OWASP 核心规则集（CRS）。在「Web 服务」的站点安全设置中选择实例即可启用；修改实例后立即生效，无需重启站点。
    </el-alert>
    <el-card shadow="never">
      <el-table :data="instances" empty-text="暂无实例">
        <el-table-column label="启用" width="70">
          <template #default="{ row }">
            <el-switch :model-value="row.enable" size="small" @change="(v: string | number | boolean) => toggle(row, Boolean(v))" />
          </template>
        </el-table-column>
        <el-table-column label="实例" min-width="160">
          <template #default="{ row }">
            <div>{{ row.name }}</div>
            <div class="meta mono">{{ row.key }}</div>
          </template>
        </el-table-column>
        <el-table-column label="模式" width="120">
          <template #default="{ row }">
            <el-tag :type="row.mode === 'block' ? 'danger' : 'warning'" size="small">{{ row.mode === 'block' ? '拦截' : '仅检测' }}</el-tag>
          </template>
        </el-table-column>
        <el-table-column label="规则" min-width="220">
          <template #default="{ row }">
            <div v-for="(r, i) in row.ruleSets" :key="i" class="meta">
              {{ r.enable ? '●' : '○' }} {{ r.type === 'crs' ? 'CRS' : '自定义' }}{{ r.remark ? '：' + r.remark : '' }}
              <span v-if="r.type === 'crs'">（{{ r.param ? r.param.split('\n').filter(Boolean).length + ' 个文件' : '全部' }}）</span>
            </div>
          </template>
        </el-table-column>
        <el-table-column label="阈值 / 等级" width="140">
          <template #default="{ row }">入 {{ row.inboundThreshold }} · 出 {{ row.outboundThreshold }} · PL{{ row.paranoia }}</template>
        </el-table-column>
        <el-table-column label="统计" width="140">
          <template #default="{ row }">
            <div>拦截 {{ row.blocked }}</div>
            <div class="meta">命中 {{ row.matched }}</div>
            <div v-if="row.error" class="err" :title="row.error">编译失败</div>
          </template>
        </el-table-column>
        <el-table-column label="操作" width="120">
          <template #default="{ row }">
            <el-button link type="primary" @click="openEditor(row)">编辑</el-button>
            <el-popconfirm title="删除该实例？引用它的站点将不再检查" @confirm="remove(row)">
              <template #reference>
                <el-button link type="danger">删除</el-button>
              </template>
            </el-popconfirm>
          </template>
        </el-table-column>
      </el-table>
    </el-card>
    <el-card shadow="never" header="拦截与命中日志" class="logs">
      <LogViewer url="/api/waf/logs" clearable />
    </el-card>

    <el-dialog v-model="editorVisible" :title="editing.key ? '编辑实例' : '添加实例'" width="760px" top="5vh">
      <el-form :model="editing" label-width="120px">
        <el-form-item label="名称"><el-input v-model="editing.name" /></el-form-item>
        <el-form-item label="模式">
          <el-radio-group v-model="editing.mode">
            <el-radio-button value="block">拦截</el-radio-button>
            <el-radio-button value="detect">仅检测（只记录日志）</el-radio-button>
          </el-radio-group>
        </el-form-item>
        <el-form-item label="偏执等级">
          <el-slider v-model="editing.paranoia" :min="1" :max="4" show-stops style="width: 240px" />
          <span class="help">越高越严格，误报也越多，建议从 1 开始</span>
        </el-form-item>
        <el-form-item label="入站阈值"><el-input-number v-model="editing.inboundThreshold" :min="1" :max="100" /></el-form-item>
        <el-form-item label="出站阈值"><el-input-number v-model="editing.outboundThreshold" :min="1" :max="100" /></el-form-item>
        <el-divider content-position="left">规则</el-divider>
        <div v-for="(r, i) in editing.ruleSets" :key="i" class="ruleset">
          <div class="ruleset-head">
            <el-switch v-model="r.enable" size="small" />
            <el-radio-group v-model="r.type" size="small">
              <el-radio-button value="crs">OWASP CRS</el-radio-button>
              <el-radio-button value="custom">自定义 SecLang</el-radio-button>
            </el-radio-group>
            <el-input v-model="r.remark" size="small" placeholder="备注" style="width: 200px" />
            <el-button link type="danger" @click="editing.ruleSets.splice(i, 1)">删除</el-button>
          </div>
          <el-select v-if="r.type === 'crs'" :model-value="crsSelected(r)" multiple filterable collapse-tags collapse-tags-tooltip
            placeholder="留空表示启用全部规则文件" style="width: 100%" @update:model-value="(v: string[]) => (r.param = v.join('\n'))">
            <el-option v-for="f in crsFiles" :key="f" :label="f" :value="f" />
          </el-select>
          <el-input v-else v-model="r.param" type="textarea" :rows="4" class="mono" placeholder='SecRule REQUEST_URI "@beginsWith /admin" "id:10001,phase:1,deny,status:403"' />
        </div>
        <el-button size="small" @click="editing.ruleSets.push({ enable: true, remark: '', type: 'crs', param: '' })">添加规则组</el-button>
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
import { api } from '../api'
import LogViewer from '../components/LogViewer.vue'

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

const instances = ref<InstanceView[]>([])
const crsFiles = ref<string[]>([])
const editorVisible = ref(false)
const saving = ref(false)
const editing = ref<Instance>(empty())
let timer: number | undefined

function empty(): Instance {
  return { name: '', enable: true, mode: 'block', paranoia: 1, inboundThreshold: 5, outboundThreshold: 4, ruleSets: [{ enable: true, remark: 'OWASP 核心规则', type: 'crs', param: '' }], remark: '' }
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
  await api.put(`/api/waf/instances/${row.key}/enable`, { enable })
  await load()
}

async function remove(row: InstanceView) {
  await api.del(`/api/waf/instances/${row.key}`)
  await load()
}

onMounted(async () => {
  crsFiles.value = await api.get('/api/waf/crs')
  await load()
  timer = window.setInterval(load, 5000)
})
onBeforeUnmount(() => window.clearInterval(timer))
</script>

<style scoped>
.intro {
  margin-bottom: 14px;
}
.meta {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.err {
  font-size: 12px;
  color: var(--el-color-danger);
}
.help {
  margin-left: 12px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.ruleset {
  border: 1px solid var(--el-border-color-lighter);
  border-radius: 6px;
  padding: 10px;
  margin-bottom: 10px;
}
.ruleset-head {
  display: flex;
  gap: 10px;
  align-items: center;
  margin-bottom: 8px;
}
.remark {
  margin-top: 16px;
}
.logs {
  margin-top: 16px;
}
</style>
