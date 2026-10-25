<template>
  <FileUsers
    v-model="users"
    :title="t('sharedFileUsers.title')"
    :desc="t('sharedFileUsers.desc')"
    :mount-summary-key="mountSummaryKey"
    :mount-name-placeholder="mountNamePlaceholder"
  >
    <template #head>
      <span class="switch-label">{{ t('sharedFileUsers.enable') }}</span>
      <el-switch v-model="enabled" :aria-label="t('sharedFileUsers.enable')" />
    </template>
    <template #notice>
      <div class="notice">
        <div>{{ smb ? t('sharedFileUsers.helpSmb') : t('sharedFileUsers.help') }}</div>
        <el-tag v-if="!enabled" size="small" type="info">{{ t('sharedFileUsers.off') }}</el-tag>
        <div v-if="loadError" class="tags">
          <el-tag size="small" type="danger">{{ t('sharedFileUsers.loadError', { error: loadError }) }}</el-tag>
        </div>
        <div v-if="enabled && shadowed.length" class="tags">
          <el-tag size="small" type="warning">{{ t('sharedFileUsers.shadowed', { names: shadowed.join(', ') }) }}</el-tag>
        </div>
        <div v-if="enabled && issues?.length" class="tags">
          <el-tag v-for="is in issues" :key="is.name" size="small" type="warning">{{ issueText(is) }}</el-tag>
        </div>
      </div>
    </template>
  </FileUsers>
</template>

<script setup lang="ts">
import { computed } from 'vue'
import { useI18n } from 'vue-i18n'
import FileUsers, { type FileUser } from './FileUsers.vue'
import type { SMBIssue } from '../composables/useSharedFileUsers'

const props = defineProps<{
  // localNames 为本服务的本地用户名，同名的共享用户只按本地用户校验
  localNames: string[]
  // smb 为 true 时按 SMB 的规则比较用户名（不区分大小写）并显示 SMB 特有的说明
  smb?: boolean
  // issues 为无法用于 SMB 的共享用户（已保存的内容），只有 SMB 页面传入
  issues?: SMBIssue[]
  // loadError 为服务端读取到的共享用户配置未通过校验的错误
  loadError?: string
  mountSummaryKey?: string
  mountNamePlaceholder?: string
}>()
const users = defineModel<FileUser[]>({ required: true })
const enabled = defineModel<boolean>('enabled', { required: true })
const { t } = useI18n()

// issueText 按原因代码返回说明，未知代码按共享名问题处理
function issueText(is: SMBIssue): string {
  switch (is.reason) {
    case 'name':
      return t('sharedFileUsers.issue.name', { name: is.name })
    case 'bcrypt':
      return t('sharedFileUsers.issue.bcrypt', { name: is.name })
    default:
      return t('sharedFileUsers.issue.share', { name: is.name })
  }
}

// shadowed 为与本地用户同名、在本服务中按本地用户校验的共享用户
const shadowed = computed(() => {
  const norm = (s: string) => (props.smb ? s.trim().toLowerCase() : s.trim())
  const local = new Set(props.localNames.map(norm))
  return users.value.map((u) => u.name.trim()).filter((n) => n && local.has(norm(n)))
})
</script>

<style scoped>
.switch-label {
  font-size: 13px;
  color: var(--el-text-color-regular);
}
.notice {
  display: flex;
  flex-direction: column;
  align-items: flex-start;
  gap: 6px;
  font-size: 12px;
  line-height: 1.6;
  color: var(--el-text-color-secondary);
  margin-bottom: 10px;
}
.tags {
  display: flex;
  flex-wrap: wrap;
  gap: 6px;
}
.tags .el-tag {
  max-width: 100%;
  height: auto;
  white-space: normal;
  line-height: 1.5;
  padding: 2px 8px;
}
</style>
