// Package systz 在系统没有标准时区文件时，从 OpenWrt 的配置中识别本机时区。
//
// Go 的 time.Local 依赖 TZ 环境变量或 /etc/localtime。OpenWrt 两者都没有：时区写在 /etc/config/system 的
// zonename（IANA 名称）和 /etc/TZ（POSIX 字符串）里，procd 启动服务时也不传 TZ，结果进程一律按 UTC 计时，
// 日志与通知时间会与系统相差数小时。IANA 时区数据由入口 import time/tzdata 内嵌，不依赖系统的 zoneinfo。
package systz

import (
	"bufio"
	"os"
	"strconv"
	"strings"
	"time"
)

// 默认读取的文件路径，测试时替换。
var (
	localtimePath = "/etc/localtime"
	uciSystemPath = "/etc/config/system"
	posixTZPath   = "/etc/TZ"
)

// Init 在 TZ 未设置且 /etc/localtime 不存在时，按 OpenWrt 配置设置 time.Local，返回识别出的时区名；
// 无法识别或无需处理时返回空串，time.Local 保持不变。需在启动其他协程之前调用。
func Init() string {
	if _, ok := os.LookupEnv("TZ"); ok {
		return ""
	}
	if _, err := os.Stat(localtimePath); err == nil {
		return ""
	}
	loc := detect()
	if loc == nil {
		return ""
	}
	time.Local = loc
	return loc.String()
}

// detect 优先使用 uci 的 zonename，其次解析 /etc/TZ。
func detect() *time.Location {
	if name := uciZonename(uciSystemPath); name != "" {
		if loc, err := time.LoadLocation(name); err == nil {
			return loc
		}
	}
	if b, err := os.ReadFile(posixTZPath); err == nil {
		return parsePOSIX(strings.TrimSpace(string(b)))
	}
	return nil
}

// uciZonename 读取 config system 段中的 option zonename，值可带单引号或双引号。
func uciZonename(path string) string {
	f, err := os.Open(path)
	if err != nil {
		return ""
	}
	defer f.Close()
	inSystem := false
	sc := bufio.NewScanner(f)
	for sc.Scan() {
		fields := strings.Fields(sc.Text())
		if len(fields) < 2 {
			continue
		}
		switch fields[0] {
		case "config":
			inSystem = fields[1] == "system"
		case "option":
			if inSystem && fields[1] == "zonename" && len(fields) >= 3 {
				return strings.Trim(strings.Join(fields[2:], " "), `'"`)
			}
		}
	}
	return ""
}

// parsePOSIX 解析不含夏令时规则的 POSIX TZ 字符串，例如 CST-8、<+0545>-5:45、UTC0。
// POSIX 的偏移与常见写法符号相反：CST-8 表示 UTC+8。带夏令时规则的字符串无法用固定偏移表示，返回 nil。
func parsePOSIX(s string) *time.Location {
	name, rest, ok := posixName(s)
	if !ok || rest == "" {
		return nil
	}
	sign := -1
	switch rest[0] {
	case '+':
		rest = rest[1:]
	case '-':
		sign = 1
		rest = rest[1:]
	}
	parts := strings.Split(rest, ":")
	if len(parts) > 3 {
		return nil
	}
	secs := 0
	unit := 3600
	for _, p := range parts {
		n, err := strconv.Atoi(p)
		if err != nil || n < 0 || p == "" {
			return nil
		}
		secs += n * unit
		unit /= 60
	}
	if secs > 24*3600 {
		return nil
	}
	return time.FixedZone(name, sign*secs)
}

// posixName 拆出 POSIX TZ 开头的时区缩写：字母形式至少 3 个字母，或用尖括号包住的任意形式。
func posixName(s string) (name, rest string, ok bool) {
	if strings.HasPrefix(s, "<") {
		end := strings.IndexByte(s, '>')
		if end < 2 {
			return "", "", false
		}
		return s[1:end], s[end+1:], true
	}
	i := 0
	for i < len(s) && (s[i] >= 'A' && s[i] <= 'Z' || s[i] >= 'a' && s[i] <= 'z') {
		i++
	}
	if i < 3 {
		return "", "", false
	}
	return s[:i], s[i:], true
}
