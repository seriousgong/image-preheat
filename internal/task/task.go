package task

import (
	"image-preheat/internal/config"
	"image-preheat/internal/preheat"
	"log"
	"sync"
	"time"
)

func StartPeriodicCheck(cache *config.ImageListCache, interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		<-ticker.C
		images := cache.GetImages()
		localImages, err := preheat.GetAllLocalImages()
		if err != nil {
			log.Printf("获取本地镜像列表失败: %v", err)
			continue
		}
		var wg sync.WaitGroup
		for _, img := range images {
			if img == "" {
				continue
			}
			if _, ok := localImages[img]; ok {
				log.Printf("本地已存在镜像: %s", img)
				continue
			}
			wg.Add(1)
			go func(image string) {
				defer wg.Done()
				if err := preheat.PreheatImageWithLimit(image); err != nil {
					log.Printf("预热镜像 %s 失败: %v", image, err)
				}
			}(img)
		}
		wg.Wait()
	}
}
