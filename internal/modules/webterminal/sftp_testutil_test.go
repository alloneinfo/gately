package webterminal

import (
	"archive/tar"
	"archive/zip"
	"io"
)

func newZipWriter(w io.Writer) *zip.Writer { return zip.NewWriter(w) }

// writeTar 写入一个含目录、文件与符号链接的 tar 包。
func writeTar(w io.Writer) {
	tw := tar.NewWriter(w)
	_ = tw.WriteHeader(&tar.Header{Name: "d/", Typeflag: tar.TypeDir, Mode: 0o755})
	_ = tw.WriteHeader(&tar.Header{Name: "d/f.txt", Typeflag: tar.TypeReg, Mode: 0o644, Size: 3})
	_, _ = tw.Write([]byte("tar"))
	_ = tw.WriteHeader(&tar.Header{Name: "d/link", Typeflag: tar.TypeSymlink, Linkname: "/etc/passwd"})
	_ = tw.Close()
}
