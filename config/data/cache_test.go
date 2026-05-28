package data

import (
	"os"
	"path/filepath"
	"testing"
	"time"
)

// TestReadFileCached_HotReloadOnChange 验证: 文件内容变化后无需手动失效即可热加载,
// 未变化时仍走内存缓存(不重复读盘)。
func TestReadFileCached_HotReloadOnChange(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "hot.yml")
	const name = "hot-reload-test"
	readDisk := func() ([]byte, error) { return os.ReadFile(path) }

	if err := os.WriteFile(path, []byte("v1"), 0o644); err != nil {
		t.Fatal(err)
	}
	b, err := readFileCached(name, path, readDisk)
	if err != nil || string(b) != "v1" {
		t.Fatalf("首次读取期望 v1, 得到 %q (err=%v)", b, err)
	}

	// 文件未变化: 应走缓存, 不应再调用 readDisk
	called := false
	b, err = readFileCached(name, path, func() ([]byte, error) { called = true; return []byte("STALE"), nil })
	if err != nil || string(b) != "v1" {
		t.Fatalf("未变化应返回缓存 v1, 得到 %q (err=%v)", b, err)
	}
	if called {
		t.Error("文件未变化时不应重新读盘")
	}

	// 修改文件(内容长度不同, 并把 mtime 往后挪, 确保变化被检测到)
	if err = os.WriteFile(path, []byte("version-2"), 0o644); err != nil {
		t.Fatal(err)
	}
	later := time.Now().Add(2 * time.Second)
	if err = os.Chtimes(path, later, later); err != nil {
		t.Fatal(err)
	}

	b, err = readFileCached(name, path, readDisk)
	if err != nil || string(b) != "version-2" {
		t.Fatalf("修改后期望热加载 version-2, 得到 %q (err=%v)", b, err)
	}

	invalidateFileCache(name)
}
