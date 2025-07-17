package docker

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
)

// DockerClient 定义 Docker 操作接口
type DockerClient interface {
	// 拉取镜像
	Pull(image string) error
	// 保存镜像到流
	Save(image string, writer io.Writer) error
	// 从流加载镜像
	Load(reader io.Reader) error
	// 获取本地镜像列表
	GetImages() (map[string]struct{}, error)
	// 检查镜像是否存在
	ImageExists(image string) (bool, error)
	// 获取镜像的所有层digest
	GetImageDigests(image string) ([]string, error)
	// 检查单个层是否存在
	CheckLayerExists(digest string) (bool, error)
	// 批量检查层是否存在
	CheckLayersExist(digests []string) (exists, missing []string, err error)
}

// CommandLineClient 基于命令行的 Docker 客户端实现
type CommandLineClient struct{}

// NewCommandLineClient 创建命令行 Docker 客户端
func NewCommandLineClient() *CommandLineClient {
	return &CommandLineClient{}
}

// Pull 拉取镜像
func (c *CommandLineClient) Pull(image string) error {
	cmd := exec.Command("docker", "pull", image)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Save 保存镜像到流
func (c *CommandLineClient) Save(image string, writer io.Writer) error {
	cmd := exec.Command("docker", "save", image)
	cmd.Stdout = writer
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// Load 从流加载镜像
func (c *CommandLineClient) Load(reader io.Reader) error {
	cmd := exec.Command("docker", "load")
	cmd.Stdin = reader
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	return cmd.Run()
}

// GetImages 获取本地镜像列表
func (c *CommandLineClient) GetImages() (map[string]struct{}, error) {
	cmd := exec.Command("docker", "images", "--format", "{{.Repository}}:{{.Tag}}")
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	images := make(map[string]struct{})
	for _, line := range strings.Split(string(output), "\n") {
		if line != "" {
			images[line] = struct{}{}
		}
	}
	return images, nil
}

// ImageExists 检查镜像是否存在
func (c *CommandLineClient) ImageExists(image string) (bool, error) {
	images, err := c.GetImages()
	if err != nil {
		return false, err
	}
	_, exists := images[image]
	return exists, nil
}

// GetImageDigests 获取镜像的所有层digest
func (c *CommandLineClient) GetImageDigests(image string) ([]string, error) {
	// 使用 docker inspect 获取镜像的RootFS.Layers（diffID）
	cmd := exec.Command("docker", "inspect", "--format={{json .RootFS.Layers}}", image)
	output, err := cmd.Output()
	if err != nil {
		return nil, err
	}

	// 解析JSON数组，获取diffID列表
	var diffIDs []string
	if err := json.Unmarshal(output, &diffIDs); err != nil {
		return nil, err
	}

	var digests []string
	for _, diffID := range diffIDs {
		diffID = strings.TrimSpace(diffID)
		if diffID == "" {
			continue
		}
		// 从diffID获取digest
		digest, err := c.getDigestFromDiffID(diffID)
		if err != nil {
			// log.Warn().Err(err).Str("diffID", diffID).Str("image", image).Msg("获取digest失败，跳过")
			continue
		}
		digests = append(digests, digest)
	}

	return digests, nil
}

// CheckLayerExists 检查单个层是否存在
func (c *CommandLineClient) CheckLayerExists(digest string) (bool, error) {
	// 构建overlay2存储路径
	diffIDPath := filepath.Join("/var/lib/docker/image/overlay2/distribution/diffid-by-digest/sha256", digest)

	// 检查diffid-by-digest文件是否存在
	if _, err := os.Stat(diffIDPath); os.IsNotExist(err) {
		return false, nil
	}

	// 读取diffID
	diffIDBytes, err := os.ReadFile(diffIDPath)
	if err != nil {
		return false, err
	}

	diffID := strings.TrimSpace(string(diffIDBytes))
	if diffID == "" {
		return false, nil
	}

	// 检查layerdb目录是否存在
	layerdbPath := filepath.Join("/var/lib/docker/image/overlay2/layerdb/sha256", diffID)
	if _, err := os.Stat(layerdbPath); os.IsNotExist(err) {
		return false, nil
	}

	return true, nil
}

// CheckLayersExist 批量检查层是否存在
func (c *CommandLineClient) CheckLayersExist(digests []string) (exists, missing []string, err error) {
	for _, digest := range digests {
		layerExists, err := c.CheckLayerExists(digest)
		if err != nil {
			return nil, nil, err
		}
		if layerExists {
			exists = append(exists, digest)
		} else {
			missing = append(missing, digest)
		}
	}
	return exists, missing, nil
}

// getDigestFromDiffID 从diffID获取对应的digest
func (c *CommandLineClient) getDigestFromDiffID(diffID string) (string, error) {
	// 只取sha256:后面的部分
	parts := strings.SplitN(diffID, ":", 2)
	if len(parts) != 2 {
		return "", fmt.Errorf("diffID格式错误: %s", diffID)
	}
	diffIDShort := parts[1]

	// 构建v2metadata-by-diffid文件路径
	metadataPath := filepath.Join("/var/lib/docker/image/overlay2/distribution/v2metadata-by-diffid/sha256", diffIDShort)

	// 读取metadata文件
	metadataBytes, err := os.ReadFile(metadataPath)
	if err != nil {
		return "", err
	}

	// 解析JSON获取digest
	var metadata []struct {
		Digest string `json:"Digest"`
	}
	if err := json.Unmarshal(metadataBytes, &metadata); err != nil {
		return "", err
	}

	if len(metadata) == 0 {
		return "", fmt.Errorf("未找到digest信息")
	}

	return metadata[0].Digest, nil
}

// 全局 Docker 客户端实例
var defaultClient DockerClient

// InitDockerClient 初始化 Docker 客户端
func InitDockerClient() error {
	defaultClient = NewCommandLineClient()
	return nil
}

// GetClient 获取默认 Docker 客户端
func GetClient() DockerClient {
	if defaultClient == nil {
		// 如果未初始化，使用默认实现
		defaultClient = NewCommandLineClient()
	}
	return defaultClient
}

// 便捷函数，直接调用默认客户端
func Pull(image string) error {
	return GetClient().Pull(image)
}

func Save(image string, writer io.Writer) error {
	return GetClient().Save(image, writer)
}

func Load(reader io.Reader) error {
	return GetClient().Load(reader)
}

func GetImages() (map[string]struct{}, error) {
	return GetClient().GetImages()
}

func ImageExists(image string) (bool, error) {
	return GetClient().ImageExists(image)
}

// 新增层状态查询便捷函数
func GetImageDigests(image string) ([]string, error) {
	return GetClient().GetImageDigests(image)
}

func CheckLayerExists(digest string) (bool, error) {
	return GetClient().CheckLayerExists(digest)
}

func CheckLayersExist(digests []string) (exists, missing []string, err error) {
	return GetClient().CheckLayersExist(digests)
}
