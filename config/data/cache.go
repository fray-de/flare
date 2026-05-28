package data

import (
	"os"
	"sync"
	"time"
)

type fileCacheEntry struct {
	data    []byte
	modTime time.Time
	size    int64
}

var (
	fileCacheMu sync.RWMutex
	fileCache   = map[string]*fileCacheEntry{}
)

// readFileCached 读取并缓存配置文件内容。
// 缓存基于文件的 mtime + size 自动失效: 直接修改磁盘上的 yml 后, 下次请求(刷新页面)即热加载,
// 无需重启容器或经设置页保存。文件未变化时走内存缓存, 仅多一次 os.Stat, 开销可忽略。
func readFileCached(name, filePath string, readDisk func() ([]byte, error)) ([]byte, error) {
	fi, statErr := os.Stat(filePath)
	fresh := func(e *fileCacheEntry) bool {
		return e != nil && statErr == nil && fi.ModTime().Equal(e.modTime) && fi.Size() == e.size
	}

	fileCacheMu.RLock()
	entry := fileCache[name]
	fileCacheMu.RUnlock()
	if fresh(entry) {
		return entry.data, nil
	}

	fileCacheMu.Lock()
	defer fileCacheMu.Unlock()
	if entry = fileCache[name]; fresh(entry) {
		return entry.data, nil
	}

	b, err := readDisk()
	if err != nil {
		return nil, err
	}
	newEntry := &fileCacheEntry{data: b}
	if statErr == nil {
		newEntry.modTime = fi.ModTime()
		newEntry.size = fi.Size()
	}
	fileCache[name] = newEntry
	return b, nil
}

func invalidateFileCache(name string) {
	fileCacheMu.Lock()
	defer fileCacheMu.Unlock()
	delete(fileCache, name)
}
