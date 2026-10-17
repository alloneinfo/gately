# 界面多语言约定

- 文案目录：`src/locales/<语言代码>/<命名空间>.json`，文件名即命名空间（如 `wol.json` 对应 `t('wol.title')`）。
- 源语言 `zh-CN`，回退语言 `en`。新增或修改文案时，至少同步更新 `zh-CN` 与 `en`，其余语言可由翻译流程补齐。
- 组件内：`const { t } = useI18n()`，模板中 `{{ t('ns.key') }}` / `:label="t('ns.key')"`。
- 非组件的 `.ts`：`import { t } from '../i18n'`，在函数内调用（不要在模块顶层求值，否则切换语言后不会刷新）。
- 插值使用 `{name}`；避免使用 `|` 复数语法，改用中性句式（如「数量：{count}」）。字面量 `{ } @ $ |` 需写成 `{'{'}` 这类转义。
- 含链接、加粗等富文本：用 `<i18n-t keypath="ns.key" tag="span">` 配合具名插槽，不用 `v-html`。
- 产品与协议名（DDNS、STUN、frp、Tailscale、Docker、WebDAV 等）保持原文。
- 后端返回的数据（错误信息、日志行、用户自填内容）不翻译。
- 校验：`node scripts/check-i18n.mjs [语言代码 ...]`，检查引用的键是否存在、各语言键集合与占位符是否与 `zh-CN` 一致。
- 通用词（保存、取消、删除……）放在 `common.json`，各页面直接引用 `t('common.save')`。
