# Image Preheat Helm Chart

Kubernetes DaemonSet 镜像预热与节点间分发系统的 Helm Chart 部署方案。

## 快速开始

### 1. 添加 Helm 仓库（可选）
```bash
helm repo add image-preheat https://your-helm-repo.com
helm repo update
```

### 2. 安装 Chart
```bash
# 使用默认配置安装
helm install image-preheat ./image-preheat

# 使用自定义配置安装
helm install image-preheat ./image-preheat \
  --set image.repository=your-registry/image-preheat \
  --set image.tag=v1.0.0 \
  --set imageList[0]=nginx:latest \
  --set imageList[1]=redis:7-alpine
```

### 3. 升级 Chart
```bash
helm upgrade image-preheat ./image-preheat \
  --set image.tag=v1.1.0
```

### 4. 卸载 Chart
```bash
helm uninstall image-preheat
```

## 配置参数

### 镜像配置
| 参数 | 描述 | 默认值 |
|------|------|--------|
| `image.repository` | 镜像仓库地址 | `your-registry/image-preheat` |
| `image.tag` | 镜像标签 | `latest` |
| `image.pullPolicy` | 镜像拉取策略 | `IfNotPresent` |

### 应用配置
| 参数 | 描述 | 默认值 |
|------|------|--------|
| `config.lockTimeout` | 分布式锁超时时间 | `5m` |
| `config.preheatConcurrency` | 预热任务并发数 | `1` |
| `config.downloadAPIConcurrency` | 下载API并发数 | `4` |
| `config.interval` | 镜像检查间隔 | `1m` |
| `config.downloadRateLimit` | 下载限速（字节/秒） | `524288000` |

### 镜像列表
```yaml
imageList:
  - "nginx:latest"
  - "redis:7-alpine"
  - "postgres:15"
```

### 资源限制
```yaml
resources:
  limits:
    cpu: 1000m
    memory: 1Gi
  requests:
    cpu: 100m
    memory: 128Mi
```

### 安全配置
```yaml
securityContext:
  runAsNonRoot: true
  runAsUser: 1000
  runAsGroup: 1000
  fsGroup: 1000
  allowPrivilegeEscalation: false
  readOnlyRootFilesystem: false
  capabilities:
    drop:
      - ALL
```

## 部署示例

### 生产环境配置
```yaml
# values-production.yaml
image:
  repository: your-registry/image-preheat
  tag: v1.0.0
  pullPolicy: Always

namespace: image-preheat

imageList:
  - "nginx:1.25-alpine"
  - "redis:7.2-alpine"
  - "postgres:15-alpine"
  - "mysql:8.0"
  - "elasticsearch:8.11.0"

config:
  preheatConcurrency: 2
  downloadAPIConcurrency: 8
  downloadRateLimit: "1048576000"  # 1GB/s

resources:
  limits:
    cpu: 2000m
    memory: 2Gi
  requests:
    cpu: 200m
    memory: 256Mi

nodeSelector:
  node-role.kubernetes.io/worker: "true"

tolerations:
  - key: "node-role.kubernetes.io/master"
    operator: "Exists"
    effect: "NoSchedule"
```

安装命令：
```bash
helm install image-preheat ./image-preheat \
  -f values-production.yaml \
  --namespace image-preheat \
  --create-namespace
```

### 开发环境配置
```yaml
# values-dev.yaml
image:
  repository: your-registry/image-preheat
  tag: dev
  pullPolicy: Always

namespace: image-preheat-dev

imageList:
  - "nginx:latest"
  - "redis:latest"

config:
  preheatConcurrency: 1
  downloadAPIConcurrency: 2
  interval: "30s"

resources:
  limits:
    cpu: 500m
    memory: 512Mi
  requests:
    cpu: 100m
    memory: 128Mi
```

## 监控配置

### 启用 ServiceMonitor
```yaml
serviceMonitor:
  enabled: true
  interval: "30s"
  scrapeTimeout: "10s"
  additionalLabels:
    release: prometheus
```

### 启用外部访问服务
```yaml
service:
  enabled: true
  type: ClusterIP
  port: 8080
```

### 启用 Ingress
```yaml
ingress:
  enabled: true
  className: nginx
  annotations:
    nginx.ingress.kubernetes.io/rewrite-target: /
  hosts:
    - host: image-preheat.example.com
      paths:
        - path: /
          pathType: Prefix
          port: 8080
```

## 故障排查

### 检查 Pod 状态
```bash
kubectl get pods -n image-preheat -l app.kubernetes.io/name=image-preheat
```

### 查看日志
```bash
kubectl logs -n image-preheat -l app.kubernetes.io/name=image-preheat
```

### 检查 ConfigMap
```bash
kubectl get configmap -n image-preheat
kubectl describe configmap image-preheat-lock -n image-preheat
```

### 访问健康检查
```bash
kubectl port-forward -n image-preheat svc/image-preheat-peers 8080:8080
curl http://localhost:8080/health
```

### 查看 Prometheus 指标
```bash
curl http://localhost:8080/metrics
```

## 注意事项

1. **Docker Socket 权限**：确保 Pod 有权限访问节点的 Docker Socket
2. **镜像仓库认证**：如需拉取私有镜像，请配置 `imagePullSecrets`
3. **资源限制**：根据节点配置调整 CPU 和内存限制
4. **网络策略**：确保 Pod 间可以正常通信（用于节点间分发）
5. **存储空间**：确保节点有足够的存储空间用于镜像缓存

## 升级指南

### 从 v0.x 升级到 v1.0
1. 备份当前配置
2. 更新 Chart 版本
3. 检查配置变更
4. 执行升级

```bash
helm upgrade image-preheat ./image-preheat \
  --set image.tag=v1.0.0 \
  --reuse-values
``` 