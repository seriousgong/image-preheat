package api

import (
	"image-preheat/internal/preheat"
	"log"

	"github.com/gin-gonic/gin"
)

// Gin 版本的镜像查询接口
func ImageCheckHandlerGin(c *gin.Context) {
	image := c.Query("image")
	if image == "" {
		c.JSON(400, gin.H{"error": "缺少镜像名参数"})
		return
	}

	localImages, err := preheat.GetAllLocalImages()
	if err != nil {
		c.JSON(500, gin.H{"error": "获取本地镜像失败"})
		return
	}
	_, exists := localImages[image]
	if exists {
		c.JSON(200, gin.H{"exists": true})
	} else {
		c.JSON(404, gin.H{"exists": false})
	}
}

// Gin 版本的镜像下载接口，通过 docker save 流式输出
func ImageDownloadHandlerGin(c *gin.Context) {
	if !preheat.AcquireDownloadAPISlotNonBlock() {
		c.JSON(429, gin.H{"error": "服务繁忙，请稍后重试"})
		return
	}
	defer preheat.ReleaseDownloadAPISlot()

	image := c.Query("image")
	if image == "" {
		c.JSON(400, gin.H{"error": "缺少镜像名参数"})
		return
	}

	c.Header("Content-Type", "application/x-tar")
	c.Header("Content-Disposition", "attachment; filename="+image+".tar")

	err := preheat.StreamImageToHTTPWithRateLimit(image, c.Writer)
	if err != nil {
		log.Printf("镜像下载失败: %v", err)
		return
	}
}

// 健康检查接口
func HealthCheckHandlerGin(c *gin.Context) {
	c.JSON(200, gin.H{"status": "healthy"})
}
