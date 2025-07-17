package api

import (
	"image-preheat/internal/config"
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

// 层状态查询接口
func LayersCheckHandlerGin(c *gin.Context) {
	var request struct {
		Image   string   `json:"image" binding:"required"`
		Digests []string `json:"digests" binding:"required"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		log.Warn().Err(err).Str("path", c.FullPath()).Msg("请求参数解析失败")
		c.JSON(400, gin.H{"error": "请求参数格式错误"})
		return
	}

	log.Info().Str("image", request.Image).Int("digest_count", len(request.Digests)).Str("path", c.FullPath()).Msg("收到层状态查询请求")

	// 参数校验
	if len(request.Digests) > config.MaxDigestsPerRequest {
		log.Warn().Str("image", request.Image).Int("digest_count", len(request.Digests)).Int("max_allowed", config.MaxDigestsPerRequest).Msg("digest数量超过限制")
		c.JSON(400, gin.H{"error": "digest数量超过限制"})
		return
	}

	// 检查镜像是否存在
	localImages, err := preheat.GetAllLocalImages()
	if err != nil {
		log.Error().Err(err).Str("image", request.Image).Msg("获取本地镜像失败")
		c.JSON(500, gin.H{"error": "获取本地镜像失败"})
		return
	}

	_, imageExists := localImages[request.Image]
	if imageExists {
		// 镜像存在，检查是否为预热镜像
		digestManager := preheat.GetPreheatedDigestManager()
		if digestManager.IsPreheatedImage(request.Image) {
			// 镜像存在且为预热镜像，所有层归入preheated_exists
			log.Info().Str("image", request.Image).Msg("镜像存在且为预热镜像，所有层归入preheated_exists")
			c.JSON(200, gin.H{
				"image":            request.Image,
				"exists":           []string{},
				"preheated_exists": request.Digests,
				"missing":          []string{},
			})
			return
		}
	}

	// 镜像不存在或不是预热镜像，逐层检查
	layerExists, layerMissing, err := preheat.CheckLayersExist(request.Digests)
	if err != nil {
		log.Error().Err(err).Str("image", request.Image).Msg("层状态查询失败")
		c.JSON(500, gin.H{"error": "层状态查询失败"})
		return
	}

	// 区分预热镜像层和普通存在层
	digestManager := preheat.GetPreheatedDigestManager()
	var preheatedExists []string
	var normalExists []string

	for _, digest := range layerExists {
		if digestManager.IsPreheatedDigest(digest) {
			preheatedExists = append(preheatedExists, digest)
		} else {
			normalExists = append(normalExists, digest)
		}
	}

	log.Info().Str("image", request.Image).Int("exists", len(normalExists)).Int("preheated_exists", len(preheatedExists)).Int("missing", len(layerMissing)).Msg("层状态查询完成")

	c.JSON(200, gin.H{
		"image":            request.Image,
		"exists":           normalExists,
		"preheated_exists": preheatedExists,
		"missing":          layerMissing,
	})
}
