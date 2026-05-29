// Flare Service Worker —— 提供"可安装 + 离线"能力
// 策略:
//  - HTML 导航请求: network-first(联网永远拿最新,断网回退最近一次缓存)——避免"改了配置不更新"
//  - 静态资源(图标/css): cache-first(命中即返回,未命中再拉并缓存)
//  - 非 GET(登录/设置保存/搜索 POST)、跨域(如 api.iconify.design)一律放行,不拦截
// 升级缓存只需修改 CACHE_VERSION。
var CACHE_VERSION = 'flare-v1';
var STATIC_CACHE = 'flare-static-' + CACHE_VERSION;
var PAGE_CACHE = 'flare-pages-' + CACHE_VERSION;

// 这些路径不缓存(自身/清单需保持最新)
var BYPASS = ['/sw.js', '/register-sw.js', '/manifest.webmanifest'];

var PRECACHE = [
  '/favicon.ico',
  '/apple-touch-icon.png',
  '/android-chrome-192x192.png',
  '/android-chrome-512x512.png',
  '/maskable-512x512.png'
];

self.addEventListener('install', function (event) {
  event.waitUntil(
    caches.open(STATIC_CACHE).then(function (cache) {
      return cache.addAll(PRECACHE).catch(function () { /* 离线安装时忽略 */ });
    }).then(function () { return self.skipWaiting(); })
  );
});

self.addEventListener('activate', function (event) {
  event.waitUntil(
    caches.keys().then(function (keys) {
      return Promise.all(keys.map(function (k) {
        if (k !== STATIC_CACHE && k !== PAGE_CACHE) { return caches.delete(k); }
        return null;
      }));
    }).then(function () { return self.clients.claim(); })
  );
});

self.addEventListener('fetch', function (event) {
  var req = event.request;
  if (req.method !== 'GET') { return; }

  var url = new URL(req.url);
  if (url.origin !== self.location.origin) { return; }       // 跨域(iconify 等)不拦
  for (var i = 0; i < BYPASS.length; i++) {
    if (url.pathname === BYPASS[i]) { return; }
  }

  // HTML 页面: network-first
  if (req.mode === 'navigate') {
    event.respondWith(
      fetch(req).then(function (res) {
        // 仅缓存正常的、未发生重定向的 200(避免缓存 302 到登录页)
        if (res && res.ok && !res.redirected) {
          var copy = res.clone();
          caches.open(PAGE_CACHE).then(function (c) { c.put(req, copy); });
        }
        return res;
      }).catch(function () {
        return caches.match(req).then(function (hit) {
          return hit || caches.match('/');
        });
      })
    );
    return;
  }

  // 静态资源: cache-first
  event.respondWith(
    caches.match(req).then(function (hit) {
      if (hit) { return hit; }
      return fetch(req).then(function (res) {
        if (res && res.ok) {
          var copy = res.clone();
          caches.open(STATIC_CACHE).then(function (c) { c.put(req, copy); });
        }
        return res;
      });
    })
  );
});
