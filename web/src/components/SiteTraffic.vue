<template>
  <div>
    <template v-if="site.type === 'proxy'">
      <el-form-item label="负载均衡">
        <el-select v-model="site.lbPolicy" class="w-200">
          <el-option label="轮询（按权重）" value="" />
          <el-option label="随机（按权重）" value="random" />
          <el-option label="IP 哈希（会话保持）" value="ip_hash" />
          <el-option label="最少连接" value="least_conn" />
        </el-select>
        <div class="form-help">多个后端时生效。后端可写成 <span class="mono">http://10.0.0.2:8080 weight=3</span>，加 <span class="mono">backup</span> 表示仅在其他后端都不可用时启用</div>
      </el-form-item>

      <el-form-item label="健康检查">
        <el-switch v-model="site.health.enable" />
        <span class="inline-help">关闭时仍会被动摘除：连续 3 次连接失败的后端暂停 30 秒</span>
      </el-form-item>
      <template v-if="site.health.enable">
        <el-form-item label="检查方式">
          <el-radio-group v-model="site.health.mode">
            <el-radio-button value="http">HTTP</el-radio-button>
            <el-radio-button value="tcp">TCP</el-radio-button>
          </el-radio-group>
          <el-input v-if="site.health.mode !== 'tcp'" v-model="site.health.path" placeholder="/healthz" class="mono w-200 gap" />
        </el-form-item>
        <el-form-item label="间隔 / 超时">
          <el-input-number v-model="site.health.interval" :min="1" :max="3600" />
          <span class="inline-help">秒</span>
          <el-input-number v-model="site.health.timeout" :min="1" :max="60" class="gap" />
          <span class="inline-help">秒</span>
        </el-form-item>
        <el-form-item label="判定阈值">
          <el-input-number v-model="site.health.fails" :min="1" :max="20" />
          <span class="inline-help">次失败判不可用</span>
          <el-input-number v-model="site.health.passes" :min="1" :max="20" class="gap" />
          <span class="inline-help">次成功恢复</span>
        </el-form-item>
      </template>

      <el-form-item label="响应缓存">
        <el-switch v-model="site.cache.enable" />
        <span class="inline-help">仅缓存无 Cookie / 认证头的 GET 请求，遵循上游 Cache-Control</span>
      </el-form-item>
      <template v-if="site.cache.enable">
        <el-form-item label="默认时长">
          <el-input-number v-model="site.cache.ttl" :min="0" :max="2592000" :step="30" />
          <span class="inline-help">秒；上游未声明缓存时使用，0 表示只缓存上游明确允许的响应</span>
        </el-form-item>
        <el-form-item label="缓存容量">
          <el-input-number v-model="site.cache.maxMb" :min="1" :max="4096" />
          <span class="inline-help">MB（内存）</span>
        </el-form-item>
      </template>
    </template>

    <el-form-item v-if="site.type !== 'redirect'" label="压缩">
      <el-switch v-model="site.compress" />
      <span class="inline-help">文本类响应启用 br / gzip（后端已压缩的响应不会重复压缩）</span>
    </el-form-item>

    <el-form-item label="限流">
      <el-switch v-model="site.rateLimit.enable" />
      <span class="inline-help">按客户端 IP</span>
    </el-form-item>
    <template v-if="site.rateLimit.enable">
      <el-form-item label="请求速率">
        <el-input-number v-model="site.rateLimit.rate" :min="0" :step="1" :precision="1" />
        <span class="inline-help">次/秒</span>
        <el-input-number v-model="site.rateLimit.burst" :min="0" class="gap" />
        <span class="inline-help">突发</span>
      </el-form-item>
      <el-form-item label="并发上限">
        <el-input-number v-model="site.rateLimit.maxConns" :min="0" />
        <span class="inline-help">单个 IP 同时处理的请求数，0 为不限</span>
      </el-form-item>
    </template>

    <el-form-item label="路径重写">
      <div class="rows">
        <div v-for="(r, i) in site.rewrites" :key="i" class="row">
          <el-input v-model="r.pattern" placeholder="正则，如 ^/old/(.*)$" class="mono flex" />
          <el-input v-model="r.to" placeholder="目标，如 /new/$1" class="mono flex" />
          <el-select v-model="r.redirect" class="w-110">
            <el-option label="内部重写" :value="false" />
            <el-option label="重定向" :value="true" />
          </el-select>
          <el-select v-if="r.redirect" v-model="r.code" class="w-80">
            <el-option v-for="c in [301, 302, 307, 308]" :key="c" :label="c" :value="c" />
          </el-select>
          <el-button text type="danger" :icon="Trash2" @click="site.rewrites.splice(i, 1)" />
        </div>
        <el-button text :icon="Plus" @click="site.rewrites.push({ pattern: '', to: '', redirect: false, code: 302 })">添加规则</el-button>
        <div class="form-help">按顺序匹配，命中第一条即停止；重写后的路径发往后端，重定向会保留查询参数</div>
      </div>
    </el-form-item>
  </div>
</template>

<script setup lang="ts">
import { Plus, Trash2 } from 'lucide-vue-next'

export interface Rewrite {
  pattern: string
  to: string
  redirect: boolean
  code: number
}

export interface TrafficFields {
  type: string
  lbPolicy: string
  health: { enable: boolean; mode: string; path: string; interval: number; timeout: number; fails: number; passes: number }
  compress: boolean
  rateLimit: { enable: boolean; rate: number; burst: number; maxConns: number }
  cache: { enable: boolean; ttl: number; maxMb: number }
  rewrites: Rewrite[]
}

const site = defineModel<TrafficFields>('site', { required: true })
</script>

<style scoped>
.rows {
  width: 100%;
  display: flex;
  flex-direction: column;
  gap: 6px;
}
.row {
  display: flex;
  gap: 6px;
  align-items: center;
  flex-wrap: wrap;
}
.flex {
  flex: 1;
  min-width: 150px;
}
.w-200 {
  width: 200px;
}
.w-110 {
  width: 110px;
}
.w-80 {
  width: 80px;
}
.gap {
  margin-left: 12px;
}
.inline-help {
  margin-left: 8px;
  font-size: 12px;
  color: var(--el-text-color-secondary);
}
</style>
