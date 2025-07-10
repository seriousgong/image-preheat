package config

import (
	"bufio"
	"os"
	"path/filepath"
	"strings"
	"sync"

	"github.com/fsnotify/fsnotify"
	"github.com/rs/zerolog/log"
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
	log.Info().Str("file", c.filePath).Msg("启动镜像列表热加载 WatchAndUpdate")
	watcher, err := fsnotify.NewWatcher()
	if err != nil {
		log.Fatal().Err(err).Msg("创建文件监视器失败")
	}
	defer watcher.Close()

	// 监听整个目录，而不仅仅是文件
	dir := filepath.Dir(c.filePath)
	if err := watcher.Add(dir); err != nil {
		log.Fatal().Err(err).Msg("添加监视路径失败")
	}

	c.load()
	log.Info().Str("file", c.filePath).Msg("镜像列表初次加载完成")

	for {
		select {
		case event, ok := <-watcher.Events:
			if !ok {
				return
			}
			if event.Op&fsnotify.Write == fsnotify.Write && event.Name == c.filePath {
				log.Info().Msg("检测到镜像列表变化，重新加载...")
				c.load()
			}
		case err, ok := <-watcher.Errors:
			if !ok {
				return
			}
			log.Error().Err(err).Msg("文件监视错误")
		}
	}
}

func (c *ImageListCache) load() {
	file, err := os.Open(c.filePath)
	if err != nil {
		log.Error().Err(err).Msg("读取镜像列表失败")
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
		log.Error().Err(err).Msg("扫描镜像列表失败")
		return
	}

	c.mu.Lock()
	c.images = images
	c.mu.Unlock()
	log.Info().Strs("images", images).Msg("镜像列表已更新")
}

func (c *ImageListCache) GetImages() []string {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return append([]string{}, c.images...)
}
