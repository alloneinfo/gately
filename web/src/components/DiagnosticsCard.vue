<template>
  <el-card shadow="never" class="diag">
    <template #header>
      <div class="head">
        <span class="title"><Stethoscope :size="16" />{{ t('diagnostics.title') }}</span>
      </div>
    </template>

    <p class="desc">{{ t('diagnostics.desc') }}</p>

    <div class="lists">
      <div class="col ok">
        <div class="col-title"><ShieldCheck :size="14" />{{ t('diagnostics.includeTitle') }}</div>
        <ul>
          <li>{{ t('diagnostics.in1') }}</li>
          <li>{{ t('diagnostics.in2') }}</li>
          <li>{{ t('diagnostics.in3') }}</li>
          <li>{{ t('diagnostics.in4') }}</li>
        </ul>
      </div>
      <div class="col no">
        <div class="col-title"><ShieldOff :size="14" />{{ t('diagnostics.excludeTitle') }}</div>
        <ul>
          <li>{{ t('diagnostics.ex1') }}</li>
          <li>{{ t('diagnostics.ex2') }}</li>
          <li>{{ t('diagnostics.ex3') }}</li>
          <li>{{ t('diagnostics.ex4', { tag: '<redacted>' }) }}</li>
        </ul>
      </div>
    </div>

    <div class="opts">
      <el-switch v-model="withLogs" />
      <div>
        <div class="opt-title">{{ t('diagnostics.withLogs') }}</div>
        <div class="meta">{{ t('diagnostics.logsHint') }}</div>
      </div>
    </div>
    <el-alert type="info" :closable="false" show-icon :title="t('diagnostics.note')" class="note" />

    <div class="ops">
      <el-button :icon="Eye" :loading="previewing" @click="openPreview">{{ t('diagnostics.preview') }}</el-button>
      <el-button type="primary" :icon="Download" :loading="downloading" @click="download">{{ t('diagnostics.download') }}</el-button>
      <el-button tag="a" :href="`${REPO}/issues/new`" target="_blank" rel="noopener" :icon="MessageSquare" class="issue-btn">{{ t('diagnostics.openIssue') }}</el-button>
    </div>

    <el-dialog v-model="previewOpen" :title="t('diagnostics.previewTitle')" width="900px" top="6vh" class="diag-dialog">
      <div class="meta hint">{{ t('diagnostics.previewHint') }}</div>
      <div class="viewer">
        <div class="files">
          <button v-for="f in files" :key="f.name" type="button" class="file" :class="{ active: f.name === current?.name }" @click="current = f">
            <FileText :size="13" />
            <span class="fname mono">{{ f.name }}</span>
            <span class="fsize">{{ formatSize(f.size) }}</span>
          </button>
        </div>
        <pre class="content mono">{{ current?.content }}</pre>
      </div>
    </el-dialog>
  </el-card>
</template>

<script setup lang="ts">
import { ref } from 'vue'
import { ElMessage } from 'element-plus'
import { useI18n } from 'vue-i18n'
import { Download, Eye, FileText, MessageSquare, ShieldCheck, ShieldOff, Stethoscope } from 'lucide-vue-next'
import { api } from '../api'

interface PreviewFile {
  name: string
  size: number
  content: string
}

const { t } = useI18n()

const REPO = 'https://github.com/wangwanjie/gately'
const withLogs = ref(true)
const previewing = ref(false)
const downloading = ref(false)
const previewOpen = ref(false)
const files = ref<PreviewFile[]>([])
const current = ref<PreviewFile>()

function query(extra = '') {
  return `?logs=${withLogs.value ? 1 : 0}${extra}`
}

function formatSize(n: number) {
  if (n < 1024) {
    return `${n} B`
  }
  return `${(n / 1024).toFixed(1)} KB`
}

async function openPreview() {
  previewing.value = true
  try {
    const res = await api.get<{ files: PreviewFile[] }>(`/api/system/diagnostics${query('&preview=1')}`)
    files.value = res.files
    current.value = res.files[0]
    previewOpen.value = true
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    previewing.value = false
  }
}

async function download() {
  downloading.value = true
  try {
    const resp = await fetch(`/api/system/diagnostics${query()}`, { credentials: 'same-origin' })
    if (!resp.ok) {
      const body = await resp.text()
      let msg = t('diagnostics.downloadFailed', { status: resp.status })
      try {
        msg = JSON.parse(body).error ?? msg
      } catch {
        // 非 JSON 错误响应保持默认提示
      }
      throw new Error(msg)
    }
    const blob = await resp.blob()
    const cd = resp.headers.get('Content-Disposition') ?? ''
    const m = /filename\*?=(?:UTF-8'')?"?([^";]+)"?/i.exec(cd)
    const name = m ? decodeURIComponent(m[1]) : `gately-diagnostics-${new Date().toISOString().slice(0, 10)}.zip`
    const url = URL.createObjectURL(blob)
    const a = document.createElement('a')
    a.href = url
    a.download = name
    document.body.appendChild(a)
    a.click()
    a.remove()
    window.setTimeout(() => URL.revokeObjectURL(url), 1000)
  } catch (e) {
    ElMessage.error((e as Error).message)
  } finally {
    downloading.value = false
  }
}
</script>

<style scoped>
.diag {
  margin-bottom: 16px;
}
.title {
  display: inline-flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}
.desc {
  margin: 0 0 14px;
  font-size: 13px;
  line-height: 1.7;
  color: var(--el-text-color-regular);
}
.lists {
  display: grid;
  grid-template-columns: repeat(2, minmax(0, 1fr));
  gap: 12px;
}
.col {
  padding: 12px 14px;
  border-radius: var(--g-radius);
  border: 1px solid var(--g-border);
  font-size: 13px;
  line-height: 1.7;
}
.col ul {
  margin: 6px 0 0;
  padding-left: 18px;
  color: var(--el-text-color-regular);
}
.col-title {
  display: flex;
  align-items: center;
  gap: 6px;
  font-weight: 600;
}
.col.ok .col-title {
  color: var(--el-color-success);
}
.col.no .col-title {
  color: var(--el-color-danger);
}
.opts {
  display: flex;
  align-items: center;
  gap: 12px;
  margin-top: 16px;
}
.opt-title {
  font-size: 14px;
  font-weight: 600;
}
.note {
  margin-top: 12px;
}
.ops {
  display: flex;
  flex-wrap: wrap;
  gap: 10px;
  margin-top: 16px;
}
.ops .el-button + .el-button {
  margin-left: 0;
}
.issue-btn {
  text-decoration: none;
}
.hint {
  margin-bottom: 10px;
}
.viewer {
  display: grid;
  grid-template-columns: 240px minmax(0, 1fr);
  gap: 12px;
  height: 62vh;
}
.files {
  display: flex;
  flex-direction: column;
  gap: 2px;
  overflow: auto;
}
.file {
  display: flex;
  align-items: center;
  gap: 6px;
  padding: 6px 8px;
  border: none;
  border-radius: 6px;
  background: transparent;
  color: var(--el-text-color-regular);
  font-size: 12px;
  cursor: pointer;
  text-align: left;
}
.file:hover {
  background: var(--el-fill-color-light);
}
.file.active {
  color: var(--el-color-primary);
  background: var(--el-color-primary-light-9);
}
.fname {
  flex: 1;
  min-width: 0;
  overflow: hidden;
  text-overflow: ellipsis;
  white-space: nowrap;
}
.fsize {
  color: var(--el-text-color-placeholder);
}
.content {
  margin: 0;
  padding: 12px;
  overflow: auto;
  font-size: 12px;
  line-height: 1.6;
  white-space: pre-wrap;
  word-break: break-all;
  border-radius: var(--g-radius);
  border: 1px solid var(--g-border);
  background: var(--el-fill-color-lighter);
}

@media (max-width: 767.98px) {
  .lists {
    grid-template-columns: 1fr;
  }
  .viewer {
    grid-template-columns: 1fr;
    grid-template-rows: auto minmax(0, 1fr);
    height: 70vh;
  }
  .files {
    max-height: 120px;
  }
}
</style>
