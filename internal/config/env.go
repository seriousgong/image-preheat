package config

import (
	"os"
	"strconv"
	"time"
)

func GetEnv(key, def string) string {
	if v := os.Getenv(key); v != "" {
		return v
	}
	return def
}

func GetEnvInt(key string, def int) int {
	if v := os.Getenv(key); v != "" {
		if n, err := strconv.Atoi(v); err == nil {
			return n
		}
	}
	return def
}

func GetEnvDuration(key string, def time.Duration) time.Duration {
	if v := os.Getenv(key); v != "" {
		if d, err := time.ParseDuration(v); err == nil {
			return d
		}
	}
	return def
}

// 统一配置项
var (
	// 当前节点名（K8s Downward API 注入），用于分布式锁
	// 环境变量：NODE_NAME，默认：""
	NodeName = GetEnv("NODE_NAME", "")

	// K8s 命名空间
	// 环境变量：K8S_NAMESPACE，默认："default"
	K8sNamespace = GetEnv("K8S_NAMESPACE", "default")

	// 分布式锁使用的 ConfigMap 名称
	// 环境变量：K8S_LOCK_CM，默认："image-preheat-lock"
	K8sLockCM = GetEnv("K8S_LOCK_CM", "image-preheat-lock")

	// 分布式锁超时时间
	// 环境变量：K8S_LOCK_TIMEOUT，默认：5分钟
	K8sLockTimeout = GetEnvDuration("K8S_LOCK_TIMEOUT", 5*time.Minute)

	// 镜像列表文件路径
	// 环境变量：IMAGE_LIST_PATH，默认："/etc/preheater/images.list"
	ImageListPath = GetEnv("IMAGE_LIST_PATH", "/etc/preheater/images.list")

	// 本节点预热任务并发数（P2P+回源总和）
	// 环境变量：PREHEAT_CONCURRENCY，默认：1
	PreheatConcurrency = GetEnvInt("PREHEAT_CONCURRENCY", 1)

	// 节点间下载 API 并发数（/images/:image/download）
	// 环境变量：DOWNLOAD_API_CONCURRENCY，默认：4
	DownloadAPIConcurrency = GetEnvInt("DOWNLOAD_API_CONCURRENCY", 4)

	// 预热周期（定时检查镜像列表）
	// 环境变量：INTERVAL，默认：1分钟
	Interval = GetEnvDuration("INTERVAL", time.Minute)

	// 拉取镜像超时时间（心跳/锁有效期）
	// 环境变量：PULLING_TIMEOUT，默认：5分钟
	PullingTimeout = GetEnvDuration("PULLING_TIMEOUT", 5*time.Minute)

	// 镜像归档挂载目录
	// 环境变量：MOUNT_DIR，默认："/etc/preheater"
	MountDir = GetEnv("MOUNT_DIR", "/etc/preheater")

	// 下载总限速（所有P2P下载总和，单位：字节/秒）
	// 环境变量：DOWNLOAD_RATE_LIMIT，默认：500*1024*1024（500MB/s）
	DownloadRateLimit = GetEnvInt("DOWNLOAD_RATE_LIMIT", 500*1024*1024)

	// 节点发现服务名称
	// 环境变量：PEER_DISCOVERY_SERVICE_NAME，默认："image-preheat-peers.default.svc.cluster.local"
	PeerDiscoveryServiceName = GetEnv("PEER_DISCOVERY_SERVICE_NAME", "image-preheat-peers.default.svc.cluster.local")

	// 节点发现间隔
	// 环境变量：PEER_DISCOVERY_INTERVAL，默认：30s
	PeerDiscoveryInterval = GetEnvDuration("PEER_DISCOVERY_INTERVAL", 30*time.Second)

	// 层状态查询相关配置

	// 层状态查询 API 并发数（/layers/check）
	// 环境变量：LAYERS_CHECK_CONCURRENCY，默认：2
	LayersCheckConcurrency = GetEnvInt("LAYERS_CHECK_CONCURRENCY", 2)
	// 层状态查询最大digest数量
	// 环境变量：MAX_DIGESTS_PER_REQUEST，默认：50
	MaxDigestsPerRequest = GetEnvInt("MAX_DIGESTS_PER_REQUEST", 50)

	// Docker存储根目录
	// 环境变量：DOCKER_ROOT_DIR，默认：/var/lib/docker
	DockerRootDir = GetEnv("DOCKER_ROOT_DIR", "/var/lib/docker")

	// Docker存储驱动类型
	// 环境变量：DOCKER_STORAGE_DRIVER，默认：overlay2"
	DockerStorageDriver = GetEnv("DOCKER_STORAGE_DRIVER", "overlay2")
)
