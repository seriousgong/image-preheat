package main

import (
	"image-preheat/internal/api"
	"image-preheat/internal/config"
	"image-preheat/internal/docker"
	"image-preheat/internal/metrics"
	"image-preheat/internal/preheat"
	"image-preheat/internal/task"
	"log"

	"github.com/gin-gonic/gin"
	"github.com/prometheus/client_golang/prometheus/promhttp"
)

func main() {
	// 初始化 Docker 客户端
	if err := docker.InitDockerClient(); err != nil {
		log.Fatalf("Docker 客户端初始化失败: %v", err)
	}

	// 初始化全局下载限速桶
	preheat.InitDownloadRateLimit(int64(config.DownloadRateLimit))

	cache := config.NewImageListCache(config.ImageListPath)
	go cache.WatchAndUpdate()

	// 启动 peer 发现服务
	go preheat.StartPeerDiscovery() // 每30秒更新一次 peers

	go task.StartPeriodicCheck(cache, config.Interval)

	if err := preheat.InitK8sLock(); err != nil {
		log.Fatalf("K8s 分布式锁初始化失败: %v", err)
	}

	metrics.InitMetrics()

	r := gin.Default()
	r.GET("/health", api.HealthCheckHandlerGin)
	r.GET("/images/check", api.ImageCheckHandlerGin)
	r.GET("/images/download", api.ImageDownloadHandlerGin)
	r.GET("/metrics", gin.WrapH(promhttp.Handler()))

	go func() {
		log.Println("Gin HTTP 服务启动于 :8080 ...")
		err := r.Run(":8080")
		if err != nil {
			log.Fatalf("Gin HTTP 服务启动失败: %v", err)
		}
	}()

	select {}
}
