package docker

import (
	"io"
	"os"
	"os/exec"
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
