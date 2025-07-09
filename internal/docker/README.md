# Docker 抽象层

## 设计目的

Docker 抽象层提供了统一的接口来操作 Docker 镜像，支持多种实现方式：

- **命令行实现** (`CommandLineClient`): 基于 `docker` 命令
- **SDK 实现** (`SDKClient`): 基于 Docker SDK (可选)

## 接口定义

```go
type DockerClient interface {
    Pull(image string) error                    // 拉取镜像
    Save(image string, writer io.Writer) error  // 保存镜像到流
    Load(reader io.Reader) error                // 从流加载镜像
    GetImages() (map[string]struct{}, error)    // 获取本地镜像列表
    ImageExists(image string) (bool, error)     // 检查镜像是否存在
}
```

## 当前实现

### CommandLineClient
- 基于 `docker` 命令行工具
- 简单可靠，依赖系统安装的 Docker
- 适合大多数场景

## 扩展实现

### 切换到 Docker SDK

如果需要使用 Docker SDK，可以创建新的实现：

1. 添加依赖：
```bash
go get github.com/docker/docker/client
```

2. 实现 SDKClient：
```go
type SDKClient struct {
    cli *client.Client
}

func (c *SDKClient) Pull(image string) error {
    ctx := context.Background()
    reader, err := c.cli.ImagePull(ctx, image, types.ImagePullOptions{})
    // ... 实现细节
}
```

3. 在 main.go 中切换：
```go
// 原来的命令行客户端
defaultClient = NewCommandLineClient()

// 切换到 SDK 客户端
sdkClient, err := NewSDKClient()
if err != nil {
    log.Fatalf("Docker SDK 客户端初始化失败: %v", err)
}
defaultClient = sdkClient
```

## 优势

1. **统一接口**: 所有 Docker 操作都通过同一接口
2. **易于测试**: 可以轻松 mock 接口进行单元测试
3. **灵活切换**: 运行时可以切换不同的实现
4. **扩展性好**: 可以轻松添加新的实现（如 containerd、CRI-O 等）

## 使用示例

```go
// 直接使用便捷函数
err := docker.Pull("nginx:latest")
images, err := docker.GetImages()
exists, err := docker.ImageExists("nginx:latest")

// 或者获取客户端实例
client := docker.GetClient()
err := client.Pull("nginx:latest")
``` 