package sysstat

import (
	"encoding/binary"
	"time"

	"golang.org/x/sys/unix"
)

// macOS 上无法在不依赖 cgo 的情况下读取 CPU 与网卡累计计数，只提供静态信息，主要便于本地开发预览。
const clockTicks = 100

func readRaw() rawSample {
	return rawSample{at: time.Now()}
}

func fillPlatform(s *Snapshot) {
	s.Kernel, _ = unix.Sysctl("kern.osrelease")
	if v, err := unix.Sysctl("kern.osproductversion"); err == nil {
		s.Platform = "macOS " + v
	}
	s.CPU.Model, _ = unix.Sysctl("machdep.cpu.brand_string")
	if tv, err := unix.SysctlTimeval("kern.boottime"); err == nil {
		s.BootTime = tv.Sec
	}
	// vm.loadavg 对应 struct loadavg { fixpt_t ldavg[3]; long fscale; }
	if raw, err := unix.SysctlRaw("vm.loadavg"); err == nil && len(raw) >= 24 {
		scale := float64(binary.LittleEndian.Uint64(raw[16:24]))
		if scale > 0 {
			for i := 0; i < 3; i++ {
				s.CPU.Load = append(s.CPU.Load, float64(binary.LittleEndian.Uint32(raw[i*4:]))/scale)
			}
		}
	}
	total, err := unix.SysctlUint64("hw.memsize")
	if err != nil || total == 0 {
		return
	}
	m := &Memory{Total: total}
	// 可用内存按空闲页 + 可清除页 + 文件缓存页近似，与活动监视器的“已使用内存”口径接近
	page, _ := unix.SysctlUint32("hw.pagesize")
	var pages uint64
	for _, k := range []string{"vm.page_free_count", "vm.page_purgeable_count", "vm.page_pageable_external_count"} {
		n, _ := unix.SysctlUint32(k)
		pages += uint64(n)
	}
	avail := pages * uint64(page)
	if avail > total {
		avail = total
	}
	m.Available = avail
	m.Used = total - avail
	s.Memory = m
}

func fillProcessOS(p *Process) {}
