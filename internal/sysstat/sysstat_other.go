//go:build !linux && !darwin

package sysstat

import "time"

// 其他平台只提供 Go 运行时与网卡地址等通用信息。
const clockTicks = 100

func readRaw() rawSample {
	return rawSample{at: time.Now()}
}

func fillPlatform(s *Snapshot) {}

func fillProcessOS(p *Process) {}
