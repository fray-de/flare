package mdi

import (
	"bytes"
	"io"
	"io/fs"
)

// seekableFS 包装一个 fs.FS，使其打开的文件实现 io.ReadSeeker。
// 原因: echo/v5 的静态文件处理(fsFile -> http.ServeContent)要求文件实现 io.ReadSeeker，
// 而 soulteary/memfs 的 File 仅实现 Read/Stat/Close(无 Seek)，直接 StaticFS 服务会返回 500。
// 文件都很小(单个 SVG 图标)，整体读入内存再用 bytes.Reader 提供 Seek 完全可接受。
type seekableFS struct {
	inner fs.FS
}

func (s seekableFS) Open(name string) (fs.File, error) {
	f, err := s.inner.Open(name)
	if err != nil {
		return nil, err
	}
	info, err := f.Stat()
	if err != nil {
		f.Close()
		return nil, err
	}
	if info.IsDir() {
		return f, nil
	}
	data, err := io.ReadAll(f)
	f.Close()
	if err != nil {
		return nil, err
	}
	return &seekableFile{Reader: bytes.NewReader(data), info: info}, nil
}

type seekableFile struct {
	*bytes.Reader
	info fs.FileInfo
}

func (f *seekableFile) Stat() (fs.FileInfo, error) { return f.info, nil }
func (f *seekableFile) Close() error               { return nil }
