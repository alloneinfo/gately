<template>
  <div class="g-card">
    <div class="g-card-head">
      <div class="g-card-title"><SlidersHorizontal :size="16" />Docker 设置</div>
      <el-button type="primary" :icon="Save" :loading="saving" :disabled="!loaded" @click="save">保存</el-button>
    </div>
    <div v-loading="loading" class="g-card-body">
      <el-form v-if="loaded" :label-position="labelPosition" label-width="110px" @submit.prevent>
        <el-form-item label="Docker 地址">
          <el-input v-model="s.host" placeholder="unix:///var/run/docker.sock" class="mono" />
          <div class="form-help">Gately 运行在容器中时需挂载 /var/run/docker.sock</div>
        </el-form-item>
        <el-form-item label="自动检查更新">
          <el-input-number v-model="s.autoCheckHours" :min="0" :max="720" />
          <span class="help">间隔小时数，0 表示不自动检查</span>
        </el-form-item>
        <el-form-item label="镜像加速">
          <div class="full">
            <div v-for="(m, idx) in s.mirrors" :key="idx" class="row">
              <el-input v-model="s.mirrors[idx]" :title="m" placeholder="镜像加速地址，例如 mirror.example.com" class="mono" />
              <el-button text type="danger" :icon="Trash2" aria-label="删除" @click="s.mirrors.splice(idx, 1)" />
            </div>
            <el-button :icon="Plus" @click="s.mirrors.push('')">添加加速地址</el-button>
            <div class="form-help">拉取失败时依次尝试这些地址，成功后自动改回原镜像名</div>
          </div>
        </el-form-item>
        <el-form-item label="私有仓库">
          <div class="full">
            <div v-for="(r, i) in s.registries" :key="i" class="row reg">
              <el-input v-model="r.server" placeholder="仓库地址 registry.example.com" class="mono" />
              <el-input v-model="r.username" placeholder="用户名" />
              <el-input v-model="r.password" type="password" show-password placeholder="密码 / 令牌" />
              <el-button text type="danger" :icon="Trash2" aria-label="删除" @click="s.registries.splice(i, 1)" />
            </div>
            <el-button :icon="Plus" @click="s.registries.push({ server: '', username: '', password: '' })">添加仓库账号</el-button>
            <div class="form-help">已保存的密码显示为 ******，保持不变即保留原值</div>
          </div>
        </el-form-item>
      </el-form>
    </div>
  </div>
</template>

<script setup lang="ts">
import { onMounted, reactive, ref } from 'vue'
import { ElMessage } from 'element-plus'
import { Plus, Save, SlidersHorizontal, Trash2 } from 'lucide-vue-next'
import { api } from '../../api'
import { useFormLayout } from '../../composables/useIsMobile'
import type { DockerSettings } from './types'

const emit = defineEmits<{ saved: [] }>()
const { labelPosition } = useFormLayout()
const s = reactive<DockerSettings>({ host: '', mirrors: [], registries: [], autoCheckHours: 0 })
const loaded = ref(false)
const loading = ref(false)
const saving = ref(false)

async function load() {
  loading.value = true
  try {
    const r = await api.get<DockerSettings>('/api/docker/settings')
    Object.assign(s, { ...r, mirrors: r.mirrors ?? [], registries: r.registries ?? [] })
    loaded.value = true
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    loading.value = false
  }
}

async function save() {
  saving.value = true
  try {
    await api.put('/api/docker/settings', {
      ...s,
      mirrors: s.mirrors.map((m) => m.trim()).filter(Boolean),
      registries: s.registries.filter((r) => r.server.trim() !== '').map((r) => ({ ...r, server: r.server.trim() })),
    })
    ElMessage.success('已保存')
    emit('saved')
    await load()
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    saving.value = false
  }
}

onMounted(load)
</script>

<style scoped>
.full {
  width: 100%;
}
.row {
  display: flex;
  gap: 8px;
  margin-bottom: 8px;
}
.row .el-input {
  flex: 1;
  min-width: 0;
}
@media (max-width: 767.98px) {
  .reg {
    flex-direction: column;
    padding: 10px;
    border: 1px solid var(--g-border);
    border-radius: var(--g-radius-sm);
  }
  .reg .el-button {
    align-self: flex-end;
  }
  .row .el-button {
    min-height: 40px;
  }
  :deep(input) {
    font-size: 16px;
  }
}
</style>
