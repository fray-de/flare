package assets

import (
	"embed"
	"fmt"
	"io/fs"
	"net/http"

	"github.com/labstack/echo/v5"

	"github.com/soulteary/flare/config/define"
)

//go:embed pwa-icons/android-chrome-192x192.png pwa-icons/android-chrome-512x512.png pwa-icons/apple-touch-icon.png pwa-icons/maskable-512x512.png
var pwaIcons embed.FS

func servePwaIcon(file string) echo.HandlerFunc {
	return func(c *echo.Context) error {
		data, err := fs.ReadFile(pwaIcons, "pwa-icons/"+file)
		if err != nil {
			return echo.ErrNotFound
		}
		c.Response().Header().Set("Cache-Control", "public, max-age=2592000")
		return c.Blob(http.StatusOK, "image/png", data)
	}
}

// manifest 随主题动态生成: background/theme 用当前主题背景色, 换主题后独立窗口配色也跟着变。
func manifestHandler(c *echo.Context) error {
	bg := define.CurrentThemeColor()
	manifest := fmt.Sprintf(`{
  "name": "Flare",
  "short_name": "Flare",
  "start_url": "/",
  "scope": "/",
  "display": "standalone",
  "orientation": "any",
  "background_color": %q,
  "theme_color": %q,
  "icons": [
    {"src": "/android-chrome-192x192.png", "sizes": "192x192", "type": "image/png", "purpose": "any"},
    {"src": "/android-chrome-512x512.png", "sizes": "512x512", "type": "image/png", "purpose": "any"},
    {"src": "/maskable-512x512.png", "sizes": "512x512", "type": "image/png", "purpose": "maskable"}
  ]
}`, bg, bg)
	c.Response().Header().Set("Cache-Control", "no-cache")
	return c.Blob(http.StatusOK, "application/manifest+json", []byte(manifest))
}

func registerPWA(e *echo.Echo) {
	e.GET("/android-chrome-192x192.png", servePwaIcon("android-chrome-192x192.png"))
	e.GET("/android-chrome-512x512.png", servePwaIcon("android-chrome-512x512.png"))
	e.GET("/maskable-512x512.png", servePwaIcon("maskable-512x512.png"))
	e.GET("/apple-touch-icon.png", servePwaIcon("apple-touch-icon.png"))
	e.GET("/manifest.webmanifest", manifestHandler)
}
