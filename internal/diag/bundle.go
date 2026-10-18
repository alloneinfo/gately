package diag

import (
	"archive/zip"
	"bytes"
	"time"
)

// File 是诊断包中的一个文件。
type File struct {
	Name string `json:"name"`
	Data []byte `json:"-"`
}

// Bundle 是待导出的诊断包内容。预览与下载使用同一份数据，保证所见即所得。
type Bundle struct {
	Files []File
}

// Add 追加一个文件。
func (b *Bundle) Add(name string, data []byte) {
	b.Files = append(b.Files, File{Name: name, Data: data})
}

// NameFor 生成诊断包文件名。
func NameFor(now time.Time) string {
	return "gately-diagnostics-" + now.UTC().Format("20060102-150405") + ".zip"
}

// Zip 打包为 zip。
func (b *Bundle) Zip(now time.Time) ([]byte, error) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for _, f := range b.Files {
		w, err := zw.CreateHeader(&zip.FileHeader{Name: f.Name, Method: zip.Deflate, Modified: now})
		if err != nil {
			return nil, err
		}
		if _, err := w.Write(f.Data); err != nil {
			return nil, err
		}
	}
	if err := zw.Close(); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}
