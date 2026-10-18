package diag

import _ "embed"

// ReadmeText 是诊断包内的说明文件，让收到 zip 的人和提交者都清楚里面有什么。
// 内容放在独立文件中：它面向全球用户中英对照，不参与后端界面文案的翻译提取。
//
//go:embed README.txt
var ReadmeText string
