# Helm Charts

本目录包含项目的 Helm Chart 部署配置。

## 目录结构

```
helm/
└── image-preheat/          # 镜像预热系统 Helm Chart
    ├── Chart.yaml          # Chart 基本信息
    ├── values.yaml         # 默认配置参数
    ├── README-helm.md      # 详细使用说明
    └── templates/          # Kubernetes 资源模板
        ├── _helpers.tpl    # 模板助手函数
        ├── configmap-image-list.yaml  # 镜像列表配置
        ├── configmap-lock.yaml        # 分布式锁配置
        ├── serviceaccount.yaml        # 服务账户
        ├── clusterrole.yaml           # 集群角色
        ├── clusterrolebinding.yaml    # 集群角色绑定
        ├── service-headless.yaml      # Headless 服务
        └── daemonset.yaml             # 主应用部署
```

## 快速使用

```bash
# 进入 Chart 目录
cd helm/image-preheat

# 安装 Chart
helm install image-preheat .

# 使用自定义配置安装
helm install image-preheat . \
  --set image.repository=your-registry/image-preheat \
  --set image.tag=v1.0.0

# 升级 Chart
helm upgrade image-preheat . --set image.tag=v1.1.0

# 卸载 Chart
helm uninstall image-preheat
```

## 详细说明

请参考 `image-preheat/README-helm.md` 获取详细的使用说明和配置参数。 