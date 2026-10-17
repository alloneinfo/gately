<template>
  <div class="g-card">
    <div class="g-card-head">
      <div class="g-card-title"><Users :size="16" />用户 <span class="g-card-desc">每个用户可访问一个目录，或挂载多个目录</span></div>
      <el-button size="small" :icon="UserPlus" @click="users.push({ name: '', password: '', root: '', readOnly: false })">添加用户</el-button>
    </div>
    <div class="g-card-body">
      <div v-if="users.length" class="u-head">
        <span>用户名</span><span>密码</span><span>目录</span><span>只读</span><span />
      </div>
      <div v-for="(u, i) in users" :key="i" class="u-item">
        <div class="u-row">
          <div class="u-cell"><label class="u-label">用户名</label><el-input v-model="u.name" /></div>
          <div class="u-cell"><label class="u-label">密码</label><el-input v-model="u.password" type="password" show-password /></div>
          <div class="u-cell u-root">
            <label class="u-label">目录</label>
            <template v-if="!u.mounts?.length">
              <el-input v-model="u.root" placeholder="/data/share" class="mono" />
              <el-button link type="primary" size="small" class="u-multi" @click="toMulti(u)">挂载多个目录</el-button>
            </template>
            <span v-else class="u-summary">{{ u.mounts.length }} 个目录，登录后按名称列出</span>
          </div>
          <div class="u-cell u-ro">
            <label class="u-label">只读</label>
            <el-switch v-if="!u.mounts?.length" v-model="u.readOnly" />
            <span v-else class="u-summary">按目录</span>
          </div>
          <div class="u-cell u-del"><el-button text type="danger" :icon="Trash2" aria-label="删除用户" @click="users.splice(i, 1)" /></div>
        </div>
        <div v-if="u.mounts?.length" class="m-list">
          <div v-for="(mt, j) in u.mounts" :key="j" class="m-row">
            <div class="u-cell"><label class="u-label">名称</label><el-input v-model="mt.name" placeholder="显示名称" /></div>
            <div class="u-cell m-path"><label class="u-label">目录</label><el-input v-model="mt.path" placeholder="/data/share" class="mono" /></div>
            <div class="u-cell u-ro"><label class="u-label">只读</label><el-switch v-model="mt.readOnly" /></div>
            <div class="u-cell u-del"><el-button text type="danger" :icon="Trash2" aria-label="删除目录" @click="removeMount(u, j)" /></div>
          </div>
          <el-button link type="primary" size="small" :icon="FolderPlus" @click="u.mounts.push({ name: '', path: '', readOnly: false })">添加目录</el-button>
        </div>
      </div>
      <el-empty v-if="!users.length" description="暂无用户" :image-size="60" />
    </div>
  </div>
</template>

<script setup lang="ts">
import { FolderPlus, Trash2, UserPlus, Users } from 'lucide-vue-next'

export interface FileMount {
  name: string
  path: string
  readOnly: boolean
}
export interface FileUser {
  name: string
  password: string
  root: string
  readOnly: boolean
  // mounts 非空时 root 与 readOnly 不再使用，根目录按名称列出各目录
  mounts?: FileMount[]
}

const users = defineModel<FileUser[]>({ required: true })

// toMulti 把单目录用户转换为多目录：已填写的目录成为第一个挂载
function toMulti(u: FileUser) {
  const name = u.root.replace(/[\\/]+$/, '').split(/[\\/]/).pop() || ''
  u.mounts = [{ name, path: u.root, readOnly: u.readOnly }]
  u.root = ''
  u.readOnly = false
}

// removeMount 删除最后一个目录时回到单目录模式，保留其目录与只读设置
function removeMount(u: FileUser, j: number) {
  const mounts = u.mounts ?? []
  if (mounts.length === 1) {
    u.root = mounts[0].path
    u.readOnly = mounts[0].readOnly
    u.mounts = []
    return
  }
  mounts.splice(j, 1)
}
</script>

<style scoped>
.u-head,
.u-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 1fr) minmax(0, 2fr) 60px 40px;
  gap: 10px;
  align-items: center;
}
.u-head {
  font-size: 12px;
  color: var(--el-text-color-secondary);
  padding-bottom: 6px;
}
.u-item {
  padding: 8px 0;
  border-top: 1px solid var(--g-border);
}
.u-label {
  display: none;
}
.u-root {
  display: flex;
  align-items: center;
  gap: 8px;
}
.u-multi {
  flex-shrink: 0;
}
.u-summary {
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
.m-list {
  margin: 8px 0 0 24px;
  padding-left: 12px;
  border-left: 2px solid var(--g-border);
}
.m-row {
  display: grid;
  grid-template-columns: minmax(0, 1fr) minmax(0, 3fr) 60px 40px;
  gap: 10px;
  align-items: center;
  margin-bottom: 6px;
}
@media (max-width: 767.98px) {
  .u-head {
    display: none;
  }
  .u-item {
    padding: 12px;
    margin-bottom: 10px;
    border: 1px solid var(--g-border);
    border-radius: 8px;
  }
  .u-row,
  .m-row {
    grid-template-columns: 1fr 1fr;
  }
  .u-label {
    display: block;
    font-size: 12px;
    color: var(--el-text-color-secondary);
    margin-bottom: 2px;
  }
  .u-root {
    grid-column: 1 / -1;
    flex-wrap: wrap;
    gap: 4px 8px;
  }
  .u-root .u-label {
    width: 100%;
    margin-bottom: 0;
  }
  .m-path {
    grid-column: 1 / -1;
  }
  .u-ro {
    display: flex;
    align-items: center;
    gap: 8px;
  }
  .u-ro .u-label {
    margin: 0;
  }
  .u-del {
    display: flex;
    justify-content: flex-end;
  }
  .m-list {
    margin-left: 0;
  }
}
</style>
