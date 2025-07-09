package config

import (
	"bufio"
	"log"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
)

type ImageListCache struct {
	mu       sync.RWMutex
	images   []string
	filePath string
}

func NewImageListCache(filePath string) *ImageListCache {
	return &ImageListCache{filePath: filePath}
}

func (c *ImageListCache) WatchAndUpdate() {
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatalf("创建文件监视器失败: %v", err)
	}
	defer watcher.Close()

	// 监听整个目录，而不仅仅是文件
	dir := filepath.Dir(c.filePath)
	if err := watcher.Add(dir); err != nil {
		log.Fatalf("添加监视路径失败: %v", err)
	}

	c.load()

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write && event.Name == c.filePath {
				log.Println("检测到镜像列表变化，重新加载...")
				c.load()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Printf("文件监视错误: %v", err)
		}
	}
}

func (c *ImageListCache) load() {
	file, err := os.Open(c.filePath)
	if err != nil {
		log.Printf("读取镜像列表失败: %v", err)
		return
	}
	defer file.Close()

	var images []string
	scanner := bufio.NewScanner(file)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line != "" && !strings.HasPrefix(line, "#") {
			images = append(images, line)
		}
	}
	if err := scanner.Err(); err != nil {
		log.Printf("扫描镜像列表失败: %v", err)
		return
	}

	c.mu.Lock()
	c.images = images
	c.mu.Unlock()
	log.Printf("镜像列表已更新: %v", images)
}

func (c *ImageListCache) GetImages() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string{}, c.images...)
}
