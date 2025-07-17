package preheat

import (
	"image-preheat/internal/docker"
	"sync"

	"github.com/rs/zerolog/log"
)

// PreheatedDigestManager 预热镜像digest管理器
type PreheatedDigestManager struct {
	mu           sync.RWMutex
	imageDigests map[string]map[string]bool // image -> digest set
}

// NewPreheatedDigestManager 创建预热镜像digest管理器
func NewPreheatedDigestManager() *PreheatedDigestManager {
	return &PreheatedDigestManager{
		imageDigests: make(map[string]map[string]bool),
	}
}

// UpdateDigests 更新指定镜像的digest信息
func (p *PreheatedDigestManager) UpdateDigests(image string) {
	digests, err := docker.GetImageDigests(image)
	if err != nil {
		log.Error().Err(err).Str("image", image).Msg("获取镜像digest失败")
		return
	}

	p.mu.Lock()
	p.imageDigests[image] = make(map[string]bool)
	for _, digest := range digests {
		p.imageDigests[image][digest] = true
	}
	p.mu.Unlock()

	log.Info().Str("image", image).Int("digest_count", len(digests)).Msg("更新预热镜像digest")
}

// IsPreheatedDigest 判断指定digest是否属于预热镜像
func (p *PreheatedDigestManager) IsPreheatedDigest(digest string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	for _, digests := range p.imageDigests {
		if digests[digest] {
			return true
		}
	}
	return false
}

// IsPreheatedImage 判断指定镜像是否为预热镜像
func (p *PreheatedDigestManager) IsPreheatedImage(image string) bool {
	p.mu.RLock()
	defer p.mu.RUnlock()

	_, exists := p.imageDigests[image]
	return exists
}

// GetPreheatedImages 获取所有预热镜像列表
func (p *PreheatedDigestManager) GetPreheatedImages() []string {
	p.mu.RLock()
	defer p.mu.RUnlock()

	images := make([]string, 0, len(p.imageDigests))
	for image := range p.imageDigests {
		images = append(images, image)
	}
	return images
}

// RemoveImage 移除指定镜像的digest信息
func (p *PreheatedDigestManager) RemoveImage(image string) {
	p.mu.Lock()
	delete(p.imageDigests, image)
	p.mu.Unlock()

	log.Info().Str("image", image).Msg("移除预热镜像digest")
}

// 全局预热镜像digest管理器实例
var preheatedDigestManager = NewPreheatedDigestManager()

// GetPreheatedDigestManager 获取全局预热镜像digest管理器
func GetPreheatedDigestManager() *PreheatedDigestManager {
	return preheatedDigestManager
}
