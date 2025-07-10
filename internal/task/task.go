package task

import (
	"image-preheat/internal/config"
	"image-preheat/internal/preheat"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

func StartPeriodicCheck(cache *config.ImageListCache, interval time.Duration) {
	log.Info().Dur("interval", interval).Msg("启动定时批量预热任务 StartPeriodicCheck")
	ticker := time.NewTicker(interval)
	defer ticker.Stop()
	for {
		<-ticker.C
		log.Info().Msg("开始新一轮批量镜像预热")
		images := cache.GetImages()
		localImages, err := preheat.GetAllLocalImages()
		if err != nil {
			log.Error().Err(err).Msg("获取本地镜像列表失败")
			continue
		}
		var wg sync.WaitGroup
		for _, img := range images {
			if img == "" {
				continue
			}
			if _, ok := localImages[img]; ok {
				log.Info().Str("image", img).Msg("本地已存在镜像")
				continue
			}
			wg.Add(1)
			go func(image string) {
				defer wg.Done()
				if err := preheat.PreheatImageWithLimit(image); err != nil {
					log.Error().Err(err).Str("image", image).Msg("预热镜像失败")
				}
			}(img)
		}
		wg.Wait()
		log.Info().Msg("本轮批量镜像预热结束")
	}
}
