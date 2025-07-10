package api

import (
	"image-preheat/internal/preheat"

	"github.com/gin-gonic/gin"
	"github.com/rs/zerolog/log"
)

// Gin 版本的镜像查询接口
func ImageCheckHandlerGin(c *gin.Context) {
	image := c.Query("image")
	log.Info().Str("image", image).Str("path", c.FullPath()).Msg("收到镜像存在性检查请求")
	if image == "" {
		log.Warn().Str("path", c.FullPath()).Msg("缺少镜像名参数")
		c.JSON(400, gin.H{"error": "缺少镜像名参数"})
		return
	}

	localImages, err := preheat.GetAllLocalImages()
	if err != nil {
		log.Error().Err(err).Str("image", image).Msg("获取本地镜像失败")
		c.JSON(500, gin.H{"error": "获取本地镜像失败"})
		return
	}
	_, exists := localImages[image]
	if exists {
		log.Info().Str("image", image).Msg("镜像存在于本地")
		c.JSON(200, gin.H{"exists": true})
	} else {
		log.Info().Str("image", image).Msg("镜像不存在于本地")
		c.JSON(404, gin.H{"exists": false})
	}
}

// Gin 版本的镜像下载接口，通过 docker save 流式输出
func ImageDownloadHandlerGin(c *gin.Context) {
	image := c.Query("image")
	log.Info().Str("image", image).Str("path", c.FullPath()).Msg("收到镜像下载请求")
	if !preheat.AcquireDownloadAPISlotNonBlock() {
		log.Warn().Str("image", image).Msg("下载接口繁忙，拒绝服务")
		c.JSON(429, gin.H{"error": "服务繁忙，请稍后重试"})
		return
	}
	defer preheat.ReleaseDownloadAPISlot()

	if image == "" {
		log.Warn().Str("path", c.FullPath()).Msg("缺少镜像名参数")
		c.JSON(400, gin.H{"error": "缺少镜像名参数"})
		return
	}

	c.Header("Content-Type", "application/x-tar")
	c.Header("Content-Disposition", "attachment; filename="+image+".tar")

	err := preheat.StreamImageToHTTPWithRateLimit(image, c.Writer)
	if err != nil {
		log.Error().Err(err).Str("image", image).Msg("镜像下载失败")
		return
	}
	log.Info().Str("image", image).Msg("镜像下载成功")
}

// 健康检查接口
func HealthCheckHandlerGin(c *gin.Context) {
	log.Debug().Str("path", c.FullPath()).Msg("健康检查请求")
	c.JSON(200, gin.H{"status": "healthy"})
}
