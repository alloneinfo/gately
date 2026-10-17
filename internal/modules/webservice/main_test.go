package webservice

import (
	"os"
	"testing"

	"github.com/wangwanjie/gately/internal/i18n"
)

// 测试断言基于简体中文文案，固定服务端语言避免受运行环境 LANG 影响。
func TestMain(m *testing.M) {
	i18n.SetServerLang("zh-CN")
	os.Exit(m.Run())
}
