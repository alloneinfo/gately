package backup

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/wangwanjie/gately/internal/backup"
	"github.com/wangwanjie/gately/internal/config"
	"github.com/wangwanjie/gately/internal/logbuf"
	"github.com/wangwanjie/gately/internal/module"
	"github.com/wangwanjie/gately/internal/modules/storage"
)

// fakeStorage 把写入保存在内存里。
type fakeStorage struct {
	files   map[string][]byte
	putErr  error
	removed []string
}

func (f *fakeStorage) MountList() []storage.MountInfo { return nil }
func (f *fakeStorage) PutFile(mount, dir, name string, data []byte) error {
	if f.putErr != nil {
		return f.putErr
	}
	f.files[dir+"/"+name] = data
	return nil
}
func (f *fakeStorage) ListNames(mount, dir string) ([]string, error) {
	var out []string
	for k := range f.files {
		if strings.HasPrefix(k, dir+"/") {
			out = append(out, strings.TrimPrefix(k, dir+"/"))
		}
	}
	return out, nil
}
func (f *fakeStorage) RemoveFile(mount, p string) error {
	f.removed = append(f.removed, p)
	delete(f.files, p)
	return nil
}

func newModule(t *testing.T, st Storage) (*Module, string) {
	t.Helper()
	dir := t.TempDir()
	s, err := config.Open(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "account.json"), []byte(`{"user":"a"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	m := New(st)
	if err := m.Init(module.Env{Store: s, Logs: logbuf.New(100, nil)}); err != nil {
		t.Fatal(err)
	}
	return m, dir
}

func TestRunLocalPrunesOldBackups(t *testing.T) {
	m, dir := newModule(t, nil)
	m.cfg.Settings.Keep = 2
	base := time.Date(2026, 1, 1, 3, 0, 0, 0, time.Local)
	for i := 0; i < 4; i++ {
		at := base.Add(time.Duration(i) * 24 * time.Hour)
		m.now = func() time.Time { return at }
		st := m.Run(false)
		if len(st.Results) != 1 || !st.Results[0].OK {
			t.Fatalf("第 %d 次备份失败: %+v", i, st.Results)
		}
	}
	entries, _ := os.ReadDir(filepath.Join(dir, localDirName))
	if len(entries) != 2 {
		t.Fatalf("应只保留 2 份，实际 %d", len(entries))
	}
	// 保留的应是最新的两份
	if entries[0].Name() != backup.Name(base.Add(48*time.Hour), false) {
		t.Fatalf("保留了错误的文件: %s", entries[0].Name())
	}
}

func TestRunEncryptedAndRemoteTarget(t *testing.T) {
	fs := &fakeStorage{files: map[string][]byte{}}
	m, dir := newModule(t, fs)
	m.cfg.Settings.Password = "pw"
	m.cfg.Targets = []Target{{Key: "t", Name: "NAS", Enable: true, Mount: "m", Dir: "bk"}}
	st := m.Run(true)
	if len(st.Results) != 2 || !st.Results[0].OK || !st.Results[1].OK {
		t.Fatalf("结果不符: %+v", st.Results)
	}
	if len(fs.files) != 1 {
		t.Fatalf("远程应写入一份: %v", fs.files)
	}
	for k, data := range fs.files {
		if !strings.HasSuffix(k, backup.ExtEncrypted) || !backup.IsEncrypted(data) {
			t.Fatalf("远程备份未加密: %s", k)
		}
		files, err := backup.Parse(data, "pw")
		if err != nil || string(files["account.json"]) != `{"user":"a"}` {
			t.Fatalf("解密后内容不符: %v %v", files, err)
		}
	}
	// 本地备份目录不应被打进备份
	if _, err := os.Stat(filepath.Join(dir, localDirName)); err != nil {
		t.Fatal(err)
	}
}

func TestRunReportsFailurePerTarget(t *testing.T) {
	fs := &fakeStorage{files: map[string][]byte{}, putErr: errors.New("磁盘已满")}
	m, _ := newModule(t, fs)
	m.cfg.Targets = []Target{{Key: "t", Name: "NAS", Enable: true, Mount: "m"}}
	st := m.Run(false)
	if !st.Results[0].OK || st.Results[1].OK || !strings.Contains(st.Results[1].Error, "磁盘已满") {
		t.Fatalf("本地应成功、远程应失败: %+v", st.Results)
	}
}

func TestRunWithoutDestination(t *testing.T) {
	m, _ := newModule(t, nil)
	m.cfg.Settings.LocalEnable = false
	st := m.Run(false)
	if len(st.Results) != 1 || st.Results[0].OK {
		t.Fatalf("没有目的地应报告失败: %+v", st.Results)
	}
}

func TestSettingsValidate(t *testing.T) {
	s := Settings{Schedule: "not cron"}
	if s.Validate() == nil {
		t.Fatal("非法表达式应被拒绝")
	}
	s = Settings{}
	if err := s.Validate(); err != nil || s.Schedule != defaultSchedule || s.Keep != defaultKeep {
		t.Fatalf("默认值不符: %+v %v", s, err)
	}
	s = Settings{Keep: 1000}
	if s.Validate() == nil {
		t.Fatal("过大的保留数量应被拒绝")
	}
}
