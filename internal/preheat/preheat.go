package preheat

import (
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"time"

	"image-preheat/internal/config"
	"image-preheat/internal/docker"
	"image-preheat/internal/metrics"

	"github.com/juju/ratelimit"
	"github.com/prometheus/client_golang/prometheus"
)

// 你需要实现 imageExistsLocally 和 preheatImage

var (
	k8sLock        *config.K8sConfigMapLock
	k8sNodeName    = config.NodeName
	k8sNamespace   = config.K8sNamespace
	k8sCMName      = config.K8sLockCM
	k8sLockTimeout = config.K8sLockTimeout
	// 全局下载限速桶
	downloadRateLimitBucket *ratelimit.Bucket
)

var maxConcurrentPreheat = config.PreheatConcurrency
var preheatSemaphore = make(chan struct{}, maxConcurrentPreheat)

func acquirePreheatSlot() { preheatSemaphore <- struct{}{} }
func releasePreheatSlot() { <-preheatSemaphore }

func preheatImageWithLimit(image string) error {
	acquirePreheatSlot()
	defer releasePreheatSlot()
	return preheatImage(image)
}

func PreheatImageWithLimit(image string) error {
	return preheatImageWithLimit(image)
}

func fileExists(path string) bool {
	info, err := os.Stat(path)
	return err == nil && !info.IsDir()
}

// 用 reader 直接流式加载镜像到 docker
func loadImageFromReader(reader io.Reader) error {
	return docker.Load(reader)
}

// 查询其他节点并直接流式加载镜像
func fetchImageFromPeers(image string) error {
	peers := GetPeerIPs() // 使用新的 peer 发现机制
	if len(peers) == 0 {
		return fmt.Errorf("没有可用的 peers")
	}

	// 尝试轮询方式
	for i := 0; i < len(peers); i++ {
		peer := peerSelector.GetNextPeer()
		if err := tryDownloadFromPeer(peer, image); err == nil {
			return nil // 成功加载
		}
	}

	// 轮询失败，尝试随机选择
	for i := 0; i < 3; i++ { // 最多尝试3次
		peer := peerSelector.GetRandomPeer()
		if err := tryDownloadFromPeer(peer, image); err == nil {
			return nil // 成功加载
		}
	}

	return fmt.Errorf("集群内无可用镜像")
}

// tryDownloadFromPeer 尝试从指定 peer 下载镜像
func tryDownloadFromPeer(peer, image string) error {
	start := time.Now()
	url := fmt.Sprintf("http://%s:8080/images/download?image=%s", peer, image)
	resp, err := http.Get(url)
	if err != nil || resp.StatusCode != http.StatusOK {
		reason := metrics.ReasonHTTPError
		if err != nil {
			reason = metrics.ReasonNetwork
		} else {
			reason = fmt.Sprintf("http_%d", resp.StatusCode)
		}
		metrics.P2PFetchFailedTotal.WithLabelValues(image, peer, reason).Inc()
		return fmt.Errorf("peer fetch failed: %v", err)
	}
	defer resp.Body.Close()
	if err := loadImageFromReader(resp.Body); err != nil {
		metrics.P2PFetchFailedTotal.WithLabelValues(image, peer, metrics.ReasonLoadError).Inc()
		return err
	}
	duration := time.Since(start).Seconds()
	metrics.P2PFetchTotal.WithLabelValues(image, peer).Inc()
	metrics.P2PFetchDuration.WithLabelValues(image, peer).Observe(duration)
	return nil
}

func pullImageFromRegistry(image string) error {
	node := config.NodeName
	metrics.RegistryPullingGauge.WithLabelValues(image, node).Set(1)
	timer := prometheus.NewTimer(metrics.RegistryPullDuration.WithLabelValues(image))
	defer func() {
		timer.ObserveDuration()
		metrics.RegistryPullingGauge.WithLabelValues(image, node).Set(0)
	}()
	err := docker.Pull(image)
	if err != nil {
		metrics.RegistryPullTotal.WithLabelValues(image, metrics.ResultFailed).Inc()
	} else {
		metrics.RegistryPullTotal.WithLabelValues(image, metrics.ResultSuccess).Inc()
	}
	return err
}

func InitK8sLock() error {
	// 验证必要的环境变量
	if k8sNodeName == "" {
		return fmt.Errorf("NODE_NAME 环境变量未设置，无法初始化K8s锁")
	}

	lock, err := config.NewK8sConfigMapLock(k8sNamespace, k8sCMName, k8sLockTimeout)
	if err != nil {
		return fmt.Errorf("初始化K8s锁失败: %v", err)
	}

	k8sLock = lock
	log.Printf("K8s锁初始化成功: namespace=%s, configmap=%s, timeout=%v, node=%s",
		k8sNamespace, k8sCMName, k8sLockTimeout, k8sNodeName)
	return nil
}

// 修改预热流程，拉取前抢锁，拉取后释放
func preheatImage(image string) error {
	// P2P
	if err := fetchImageFromPeers(image); err == nil {
		metrics.ImagePreheatTotal.WithLabelValues(image, metrics.SourceP2P).Inc()
		return nil
	}
	// 回源前分布式锁抢占
	if k8sLock == nil || k8sNodeName == "" {
		log.Printf("警告：K8s锁未配置，跳过镜像拉取: %s (NODE_NAME=%s, k8sLock=%v)", image, k8sNodeName, k8sLock != nil)
		return fmt.Errorf("K8s锁未正确配置，无法安全拉取镜像")
	}
	acquired, err := k8sLock.TryAcquireLock(image, k8sNodeName)
	if err != nil {
		log.Printf("获取锁失败: %v", err)
		return err
	}
	if !acquired {
		log.Printf("有其他节点正在拉取镜像: %s，跳过本次拉取", image)
		return nil
	}
	// 启动心跳 goroutine
	stopCh := make(chan struct{})
	go func() {
		ticker := time.NewTicker(k8sLockTimeout / 3)
		defer ticker.Stop()
		for {
			select {
			case <-ticker.C:
				_ = k8sLock.RefreshLock(image, k8sNodeName)
			case <-stopCh:
				return
			}
		}
	}()
	defer close(stopCh)
	defer k8sLock.ReleaseLock(image, k8sNodeName)
	// 回源拉取
	err = pullImageFromRegistry(image)
	if err == nil {
		metrics.ImagePreheatTotal.WithLabelValues(image, metrics.SourceRegistry).Inc()
		return nil
	}
	metrics.ImagePreheatFailedTotal.WithLabelValues(image, metrics.ResultFailed).Inc()
	return err
}

func getAllLocalImages() (map[string]struct{}, error) {
	return docker.GetImages()
}

var maxConcurrentDownloadsAPI = config.DownloadAPIConcurrency
var downloadAPISemaphore = make(chan struct{}, maxConcurrentDownloadsAPI)

func releaseDownloadAPISlot() { <-downloadAPISemaphore }

func GetAllLocalImages() (map[string]struct{}, error) {
	return getAllLocalImages()
}

func FileExists(path string) bool {
	return fileExists(path)
}

func AcquireDownloadAPISlotNonBlock() bool {
	select {
	case downloadAPISemaphore <- struct{}{}:
		return true
	default:
		return false
	}
}

func ReleaseDownloadAPISlot() {
	releaseDownloadAPISlot()
}

// 流式下载镜像到 HTTP 响应
func StreamImageToHTTP(image string, writer io.Writer) error {
	// 检查镜像是否存在
	exists, err := docker.ImageExists(image)
	if err != nil {
		return fmt.Errorf("获取本地镜像列表失败: %v", err)
	}
	if !exists {
		return fmt.Errorf("镜像不存在: %s", image)
	}

	// 执行 docker save 并流式输出
	return docker.Save(image, writer)
}

// GetCurrentDownloadCount 获取当前下载并发数
func GetCurrentDownloadCount() int {
	return len(downloadAPISemaphore)
}

// GetMaxDownloadConcurrency 获取最大下载并发数
func GetMaxDownloadConcurrency() int {
	return cap(downloadAPISemaphore)
}

// 初始化下载限速桶
func InitDownloadRateLimit(bytesPerSec int64) {
	downloadRateLimitBucket = ratelimit.NewBucketWithRate(float64(bytesPerSec), bytesPerSec)
}

// 获取限速 reader
func RateLimitedReader(r io.Reader) io.Reader {
	if downloadRateLimitBucket == nil {
		return r
	}
	return ratelimit.Reader(r, downloadRateLimitBucket)
}

// 流式下载镜像到 HTTP 响应（带限速）
func StreamImageToHTTPWithRateLimit(image string, writer io.Writer) error {
	exists, err := docker.ImageExists(image)
	if err != nil {
		return fmt.Errorf("获取本地镜像列表失败: %v", err)
	}
	if !exists {
		return fmt.Errorf("镜像不存在: %s", image)
	}

	pr, pw := io.Pipe()
	go func() {
		defer pw.Close()
		_ = docker.Save(image, pw)
	}()

	_, err = io.Copy(writer, RateLimitedReader(pr))
	return err
}
