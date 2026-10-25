import { ref } from 'vue'
import { api } from '../api'
import type { FileUser } from '../components/FileUsers.vue'

// SMBIssue 是无法用于 SMB 的共享用户及原因代码（name / bcrypt / share），只有 SMB 的接口返回
export interface SMBIssue {
  name: string
  reason: string
}

// useSharedFileUsers 读取与保存 WebDAV、FTP、SMB 共用的文件服务共享用户。
// apiBase 为所在服务的接口前缀（如 /api/webdav），各服务读写的是同一份列表，按服务区分只是为了沿用各模块的访问权限。
export function useSharedFileUsers(apiBase: string) {
  const users = ref<FileUser[]>([])
  const issues = ref<SMBIssue[]>([])
  // loadError 为服务端读取到的共享用户配置未通过校验的错误，此时共享用户都不能登录，保存一份有效的列表后恢复
  const loadError = ref('')
  // savedUsers 为已保存的共享用户（不随表单编辑变化），用于展示连接地址等以已保存配置为准的内容
  const savedUsers = ref<FileUser[]>([])
  // saved 为最近一次读取或保存的内容，未修改时页面保存不再提交共享用户
  let saved = ''

  function apply(r: { users: FileUser[] | null; issues?: SMBIssue[]; error?: string }) {
    users.value = (r.users ?? []).map((u) => ({ ...u, mounts: u.mounts ?? [] }))
    issues.value = r.issues ?? []
    loadError.value = r.error ?? ''
    saved = JSON.stringify(users.value)
    savedUsers.value = JSON.parse(saved)
  }

  async function load() {
    apply(await api.get(`${apiBase}/shared-users`))
  }

  // save 只在内容有变化时提交；返回是否提交过
  async function save(): Promise<boolean> {
    if (JSON.stringify(users.value) === saved) {
      return false
    }
    apply(await api.put(`${apiBase}/shared-users`, { users: users.value }))
    return true
  }

  // saveWith 保存共享用户与本服务的设置 s（putSettings 提交服务设置）。
  // 只靠共享用户启用本服务时，服务设置的校验要看到已保存的共享用户，先保存共享用户；
  // 其他情况先保存服务设置，服务设置校验失败时共享用户保持不变，避免只保存了一半
  async function saveWith<T>(s: { enable: boolean; sharedUsers: boolean; users: unknown[] }, putSettings: () => Promise<T>): Promise<T> {
    if (s.enable && s.users.length === 0 && s.sharedUsers && users.value.length > 0) {
      await save()
      return await putSettings()
    }
    const r = await putSettings()
    await save()
    return r
  }

  return { users, savedUsers, issues, loadError, load, save, saveWith }
}
