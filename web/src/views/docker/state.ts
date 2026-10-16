// 各标签页共享的 Docker 连接状态：任意标签请求失败时汇总到这里，由外壳统一提示。
import { ref } from 'vue'
import type { DockerInfo } from './types'

export const dockerError = ref('')
export const dockerInfo = ref<DockerInfo>()

// reportError 记录最近一次连接类错误；成功时传空字符串清除。
export function reportError(msg: string) {
  dockerError.value = msg
}
