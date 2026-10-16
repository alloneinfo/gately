// 后台任务跟踪：提交任务后按 1 秒间隔轮询 /tasks/{id}，状态全局共享，供 TaskPanel 展示。
import { reactive } from 'vue'
import { ElMessage } from 'element-plus'
import { api } from '../../api'
import type { DockerTask } from './types'

export interface TrackedTask extends DockerTask {
  // 用户是否已关闭该条浮条
  dismissed: boolean
}

export const trackedTasks = reactive<TrackedTask[]>([])
const callbacks = new Map<string, (t: DockerTask) => void>()
const timers = new Map<string, number>()

async function poll(id: string) {
  try {
    const t = await api.get<DockerTask>(`/api/docker/tasks/${id}`)
    const cur = trackedTasks.find((x) => x.id === id)
    if (cur) {
      Object.assign(cur, t)
    }
    if (t.status === 'running') {
      timers.set(id, window.setTimeout(() => poll(id), 1000))
      return
    }
    timers.delete(id)
    const cb = callbacks.get(id)
    callbacks.delete(id)
    if (t.status === 'error') {
      ElMessage.error(t.error || `${t.title} 失败`)
    }
    if (cb) {
      cb(t)
    }
  } catch (e) {
    // 轮询失败（多为网络抖动）时稍后重试，任务不存在则放弃
    const msg = (e as Error).message
    if (/不存在|not found|404/i.test(msg)) {
      timers.delete(id)
      const cur = trackedTasks.find((x) => x.id === id)
      if (cur) {
        cur.status = 'error'
        cur.error = msg
      }
      return
    }
    timers.set(id, window.setTimeout(() => poll(id), 2000))
  }
}

// trackTask 开始跟踪一个任务，结束（成功或失败）后调用 onEnd。
export function trackTask(id: string, title: string, onEnd?: (t: DockerTask) => void) {
  trackedTasks.unshift({ id, kind: '', title, status: 'running', started: new Date().toISOString(), lines: [], dismissed: false })
  if (trackedTasks.length > 5) {
    trackedTasks.length = 5
  }
  if (onEnd) {
    callbacks.set(id, onEnd)
  }
  poll(id)
}

// 恢复页面上仍在运行的任务（刷新页面后继续展示进度）。
export async function resumeRunningTasks() {
  try {
    const list = await api.get<DockerTask[]>('/api/docker/tasks')
    for (const t of list ?? []) {
      if (t.status === 'running' && !trackedTasks.some((x) => x.id === t.id)) {
        trackTask(t.id, t.title)
      }
    }
  } catch {
    // 无法连接 Docker 时由主页面给出提示
  }
}
