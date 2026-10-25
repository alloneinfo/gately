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
  // savedUsers 为已保存的共享用户（不随表单编辑变化），用于展示连接地址等以已保存配置为准的内容
  const savedUsers = ref<FileUser[]>([])
  // saved 为最近一次读取或保存的内容，未修改时页面保存不再提交共享用户
  let saved = ''

  function apply(r: { users: FileUser[] | null; issues?: SMBIssue[] }) {
    users.value = (r.users ?? []).map((u) => ({ ...u, mounts: u.mounts ?? [] }))
    issues.value = r.issues ?? []
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

  return { users, savedUsers, issues, load, save }
}
