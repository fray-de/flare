// 注册 Service Worker(外部脚本以适配 CSP script-src 'self')
(function () {
  if (!('serviceWorker' in navigator)) { return; }
  window.addEventListener('load', function () {
    var swUrl = '/sw.js';
    // 兼容 require-trusted-types-for 'script': 若启用 Trusted Types 则用策略包装 URL
    try {
      if (self.trustedTypes && self.trustedTypes.createPolicy) {
        var policy = self.trustedTypes.createPolicy('flare-sw', {
          createScriptURL: function (s) { return s; }
        });
        swUrl = policy.createScriptURL('/sw.js');
      }
    } catch (e) { /* 忽略,回退为普通字符串 */ }
    navigator.serviceWorker.register(swUrl).catch(function () { /* 注册失败不影响站点 */ });
  });
})();
