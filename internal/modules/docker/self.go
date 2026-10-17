package docker

import (
	"os"
	"regexp"
	"strings"
	"sync"
)

// 本文件识别 Gately 自身所在的容器，避免在界面上误停、误删自己。

var (
	selfOnce sync.Once
	selfCID  string
	selfHost string
)

var containerIDPattern = regexp.MustCompile(`[0-9a-f]{64}`)

// detectSelf 读取当前进程所在的容器 ID（64 位十六进制）与主机名；不在容器中时 ID 为空。
func detectSelf() {
	selfHost, _ = os.Hostname()
	// 优先从 mountinfo / cgroup 中提取完整容器 ID，兼容自定义主机名
	for _, f := range []string{"/proc/self/mountinfo", "/proc/self/cgroup"} {
		data, err := os.ReadFile(f)
		if err != nil {
			continue
		}
		for _, line := range strings.Split(string(data), "\n") {
			if !strings.Contains(line, "docker") && !strings.Contains(line, "containers") {
				continue
			}
			if id := containerIDPattern.FindString(line); id != "" {
				selfCID = id
				return
			}
		}
	}
}

// isSelfContainer 判断给定容器 ID 是否为运行 Gately 的容器。
func isSelfContainer(id string) bool {
	selfOnce.Do(detectSelf)
	if id == "" {
		return false
	}
	if selfCID != "" {
		return id == selfCID
	}
	// 退回到默认主机名（容器 ID 前 12 位）
	if _, err := os.Stat("/.dockerenv"); err != nil {
		return false
	}
	return len(selfHost) >= 12 && strings.HasPrefix(id, selfHost)
}
