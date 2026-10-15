//go:build !unix

package webterminal

import "errors"

func startLocal(LocalConfig, int, int) (session, error) {
	return nil, errors.New("当前系统不支持本机 Shell")
}
