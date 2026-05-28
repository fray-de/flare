package mdi

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/soulteary/flare/config/define"
)

// setupIconifyTest 把缓存目录指向临时目录，避免触网与污染工作目录
func setupIconifyTest(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	iconifyCacheDir = dir
	iconifyMu.Lock()
	iconifyRendered = make(map[string]string)
	iconifyMu.Unlock()
	return dir
}

func TestGetIconifyIcon_InvalidNameReturnsEmpty(t *testing.T) {
	setupIconifyTest(t)
	cases := []string{
		"nocolon",           // 没有冒号不该走到这里，但单独调用也应安全
		"mdi:",              // name 为空
		":home",             // prefix 为空
		"mdi:../etc/passwd", // 路径穿越
		"mdi:home/evil",     // 非法字符
		"MDI:Home",          // 大写不匹配正则
		"mdi:home space",    // 空格
		"mdi:-home",         // 以连字符开头
	}
	for _, name := range cases {
		if got := getIconifyIcon(name); got != _EMPTY_ICON {
			t.Errorf("getIconifyIcon(%q) = %q, 期望空图标", name, got)
		}
	}
	// 非法名字不应在磁盘留下任何文件
	entries, _ := os.ReadDir(iconifyCacheDir)
	if len(entries) != 0 {
		t.Errorf("非法名字不应写入缓存文件，却发现 %d 个", len(entries))
	}
}

func TestGetIconifyIcon_DiskCacheHitImg(t *testing.T) {
	dir := setupIconifyTest(t)
	define.AppFlags.EnableMinimumRequest = false
	define.ThemeCurrent = "white"

	// 预置磁盘缓存，命中后不应触网
	fileName := define.ThemeCurrent + "-simple-icons-synology.svg"
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`<svg></svg>`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := getIconifyIcon("simple-icons:synology")
	want := `<img src="` + _ICONIFY_WEB_URI + "/" + fileName + `" width="68" height="68" alt="">`
	if got != want {
		t.Errorf("命中磁盘缓存返回 %q，期望 %q", got, want)
	}

	// 第二次应命中内存渲染缓存，结果一致
	if got2 := getIconifyIcon("simple-icons:synology"); got2 != want {
		t.Errorf("内存缓存返回 %q，期望 %q", got2, want)
	}
}

func TestGetIconifyIcon_DiskCacheHitInlineMini(t *testing.T) {
	dir := setupIconifyTest(t)
	define.AppFlags.EnableMinimumRequest = true
	t.Cleanup(func() { define.AppFlags.EnableMinimumRequest = false })

	// mini 模式缓存文件名不含 theme
	fileName := "mdi-home.svg"
	if err := os.WriteFile(filepath.Join(dir, fileName), []byte(`<svg viewBox="0 0 24 24"><path fill="currentColor" d="M1"/></svg>`), 0o644); err != nil {
		t.Fatal(err)
	}

	got := getIconifyIcon("mdi:home")
	if !strings.HasPrefix(got, "<svg") {
		t.Errorf("mini 模式应返回内联 SVG，得到 %q", got)
	}
	if !strings.Contains(got, "var(--color-primary)") {
		t.Errorf("mini 模式内联 SVG 应注入主题色变量，得到 %q", got)
	}
}

func TestInjectCurrentColor(t *testing.T) {
	in := `<svg viewBox="0 0 24 24"><path d="M1"/></svg>`
	out := injectCurrentColor(in)
	if !strings.Contains(out, `<svg viewBox="0 0 24 24" style="color: var(--color-primary);">`) {
		t.Errorf("注入结果不符合预期: %q", out)
	}
	// 无 svg 时原样返回
	if injectCurrentColor("not svg") != "not svg" {
		t.Error("非 SVG 内容应原样返回")
	}
}
