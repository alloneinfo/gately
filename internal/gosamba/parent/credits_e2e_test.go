// Gately 修改：仅在 Linux 与 macOS 编译（构建约束）。

//go:build smbclient_e2e && (linux || darwin)

// Package parent — credit window E2E tests.
//
// The go-smb2 multi-credit test (TestE2E_Credits_LargeIO) has been removed in
// favour of the smbclient 64 MiB large-file round-trip in
// credits_smbclient_e2e_test.go, which proves the same credit-window behaviour
// without requiring the github.com/cloudsoda/go-smb2 package.
package parent
