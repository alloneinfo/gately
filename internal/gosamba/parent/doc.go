// Gately 新增：不带构建约束的占位文件。
//
// 包内其余文件只在 Linux 与 macOS 编译（上游只支持这两个系统作为服务端），
// 本文件保证其他系统上包仍然有可编译的文件，go build/vet ./... 不会报 "no Go files"。

package parent
