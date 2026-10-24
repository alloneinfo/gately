//go:build !linux && !darwin

package smb

import (
	"errors"

	"github.com/wangwanjie/gately/internal/logbuf"
)

// supported 表示当前系统可以运行 SMB 服务。内置的 SMB 实现只支持 Linux 与 macOS。
const supported = false

type server struct{}

func startServer(Settings, *logbuf.Buffer) (*server, error) {
	return nil, errors.New("当前系统不支持 SMB 服务，仅支持 Linux 与 macOS")
}

func (s *server) Addr() string { return "" }

func (s *server) Close() {}
