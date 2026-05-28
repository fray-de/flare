package mdi

import (
	"errors"
	"io"
	"log"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"sync"
	"time"

	"golang.org/x/sync/singleflight"

	"github.com/soulteary/flare/config/define"
)

const _ICONIFY_DIR_NAME = "iconify"
const _ICONIFY_WEB_URI = "/assets/iconify"
const _ICONIFY_API_BASE = "https://api.iconify.design"

// prefix 与 name 都必须满足该正则，避免脏字符串被拼进 URL 或造成路径穿越
var iconifySegment = regexp.MustCompile(`^[a-z0-9]+(-[a-z0-9]+)*$`)

var (
	iconifyCacheDir   string
	iconifyMu         sync.Mutex
	iconifyRendered   map[string]string
	iconifyGroup      singleflight.Group
	iconifyHTTPClient = &http.Client{Timeout: 5 * time.Second}
)

// initIconify 在数据目录(即用户挂载的工作目录)下建立持久化缓存目录
func initIconify() error {
	dir, err := os.Getwd()
	if err != nil {
		dir = "."
	}
	iconifyCacheDir = filepath.Join(dir, _ICONIFY_DIR_NAME)
	if err := os.MkdirAll(iconifyCacheDir, 0o755); err != nil {
		return err
	}
	iconifyMu.Lock()
	iconifyRendered = make(map[string]string)
	iconifyMu.Unlock()
	return nil
}

// getIconifyIcon 处理形如 "prefix:name" 的 Iconify 图标，按需拉取 + 落盘缓存
func getIconifyIcon(name string) string {
	prefix, iconName, ok := strings.Cut(name, ":")
	if !ok || !iconifySegment.MatchString(prefix) || !iconifySegment.MatchString(iconName) {
		return _EMPTY_ICON
	}

	// EnableMinimumRequest 模式沿用内联 SVG + currentColor(由 CSS 变量着色)，与内置 MDI 风格一致，
	// 且与主题色无关，缓存 key 不含 theme；否则把主题色烘焙进 SVG 并按 {theme}-{prefix}-{name} 缓存
	mini := define.AppFlags.EnableMinimumRequest
	var cacheKey, fileName string
	if mini {
		cacheKey = prefix + ":" + iconName
		fileName = prefix + "-" + iconName + ".svg"
	} else {
		cacheKey = define.ThemeCurrent + ":" + prefix + ":" + iconName
		fileName = define.ThemeCurrent + "-" + prefix + "-" + iconName + ".svg"
	}

	iconifyMu.Lock()
	if v, hit := iconifyRendered[cacheKey]; hit {
		iconifyMu.Unlock()
		return v
	}
	iconifyMu.Unlock()

	// singleflight 保证同一图标并发只拉取一次
	v, doErr, _ := iconifyGroup.Do(cacheKey, func() (interface{}, error) {
		iconifyMu.Lock()
		if cached, hit := iconifyRendered[cacheKey]; hit {
			iconifyMu.Unlock()
			return cached, nil
		}
		iconifyMu.Unlock()

		filePath := filepath.Join(iconifyCacheDir, fileName)
		if data, readErr := os.ReadFile(filePath); readErr == nil {
			return renderIconify(mini, fileName, data), nil
		}

		data, err := fetchIconify(prefix, iconName, mini)
		if err != nil {
			log.Println("拉取 Iconify 图标出错:", name, err)
			return _EMPTY_ICON, nil
		}
		if writeErr := os.WriteFile(filePath, data, 0o644); writeErr != nil {
			log.Println("缓存 Iconify 图标出错:", writeErr)
		}
		return renderIconify(mini, fileName, data), nil
	})

	if doErr != nil {
		return _EMPTY_ICON
	}
	rendered, ok := v.(string)
	if !ok || rendered == "" {
		return _EMPTY_ICON
	}

	// 仅缓存成功渲染的结果，失败时下次仍可重试
	iconifyMu.Lock()
	iconifyRendered[cacheKey] = rendered
	iconifyMu.Unlock()
	return rendered
}

func fetchIconify(prefix, iconName string, mini bool) ([]byte, error) {
	endpoint := _ICONIFY_API_BASE + "/" + prefix + "/" + iconName + ".svg"
	if !mini {
		// 拿到的是具体颜色值，交给 Iconify 把 currentColor 替换为主题色
		if color := strings.TrimSpace(define.ThemePrimaryColor); color != "" {
			endpoint += "?color=" + url.QueryEscape(color)
		}
	}

	resp, err := iconifyHTTPClient.Get(endpoint)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, errors.New("iconify api 状态码 " + resp.Status)
	}
	body, err := io.ReadAll(io.LimitReader(resp.Body, 1<<20))
	if err != nil {
		return nil, err
	}
	if !strings.Contains(string(body), "<svg") {
		return nil, errors.New("iconify api 返回内容不是 SVG")
	}
	return body, nil
}

func renderIconify(mini bool, fileName string, data []byte) string {
	if mini {
		return injectCurrentColor(string(data))
	}
	return `<img src="` + _ICONIFY_WEB_URI + "/" + fileName + `" width="68" height="68" alt="">`
}

// injectCurrentColor 给内联 SVG 的根元素加上 color 样式，
// 使其内部的 fill="currentColor" 解析为主题色(与内置 MDI 内联分支保持一致的取色来源)
func injectCurrentColor(svg string) string {
	idx := strings.Index(svg, "<svg")
	if idx < 0 {
		return svg
	}
	end := strings.Index(svg[idx:], ">")
	if end < 0 {
		return svg
	}
	end += idx
	return svg[:end] + ` style="color: var(--color-primary);"` + svg[end:]
}
