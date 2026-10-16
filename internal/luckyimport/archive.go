package luckyimport

import (
	"archive/zip"
	"bytes"
	"errors"
	"fmt"
	"io"
	"path"
	"strings"
)

// 压缩包解压限制，防止压缩炸弹
const (
	maxArchiveEntries = 256
	maxEntrySize      = 32 << 20
	maxArchiveTotal   = 128 << 20
)

// Entry 是待导入的一个配置文件。
type Entry struct {
	Name string
	Data []byte
}

// IsArchive 判断数据是否为 zip 压缩包（Lucky「备份配置」导出的整站配置）。
func IsArchive(data []byte) bool {
	return len(data) >= 4 && bytes.Equal(data[:4], []byte("PK\x03\x04"))
}

// ExpandArchive 解出 zip 中的 .lkcf / .json 配置文件，忽略目录与其他文件。
// 文件名带上压缩包名作为前缀，便于在结果中定位。
func ExpandArchive(name string, data []byte) ([]Entry, error) {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return nil, fmt.Errorf("解析压缩包失败: %w", err)
	}
	if len(zr.File) > maxArchiveEntries {
		return nil, fmt.Errorf("压缩包内文件过多（%d 个）", len(zr.File))
	}
	var (
		out   []Entry
		total int64
	)
	for _, f := range zr.File {
		if f.FileInfo().IsDir() {
			continue
		}
		base := path.Base(f.Name)
		// 跳过 macOS 生成的元数据文件
		if strings.HasPrefix(f.Name, "__MACOSX/") || strings.HasPrefix(base, "._") {
			continue
		}
		ext := strings.ToLower(path.Ext(base))
		if ext != ".lkcf" && ext != ".json" {
			continue
		}
		rc, err := f.Open()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		b, err := io.ReadAll(io.LimitReader(rc, maxEntrySize+1))
		rc.Close()
		if err != nil {
			return nil, fmt.Errorf("%s: %w", f.Name, err)
		}
		if len(b) > maxEntrySize {
			return nil, fmt.Errorf("%s: 文件过大", f.Name)
		}
		total += int64(len(b))
		if total > maxArchiveTotal {
			return nil, errors.New("压缩包解压后过大")
		}
		out = append(out, Entry{Name: name + "/" + f.Name, Data: b})
	}
	if len(out) == 0 {
		return nil, errors.New("压缩包中没有找到 Lucky 配置文件（.lkcf）")
	}
	return out, nil
}
