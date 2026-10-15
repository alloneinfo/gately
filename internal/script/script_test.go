package script

import (
	"runtime"
	"strings"
	"testing"
	"time"
)

func TestRunVars(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	out, err := Run(`echo "$ipAddr|$GATELY_IPADDR"`, map[string]string{"ipAddr": "1.2.3.4"})
	if err != nil || strings.TrimSpace(string(out)) != "1.2.3.4|1.2.3.4" {
		t.Fatalf("%q %v", out, err)
	}
}

// 后台子进程持有输出管道时，超时后仍应及时返回
func TestRunTimeoutKillsGroup(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip()
	}
	old := Timeout
	Timeout = 300 * time.Millisecond
	defer func() { Timeout = old }()
	start := time.Now()
	_, err := Run(`sleep 30 & sleep 30`, nil)
	if err == nil {
		t.Fatal("超时应返回错误")
	}
	if d := time.Since(start); d > 5*time.Second {
		t.Fatalf("超时后未及时返回: %v", d)
	}
}
