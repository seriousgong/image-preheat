package config

import (
	"context"
	"encoding/json"
	"time"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/client-go/kubernetes"
	"k8s.io/client-go/rest"
)

type K8sLockInfo struct {
	Image     string    `json:"image"`
	Node      string    `json:"node"`
	Timestamp time.Time `json:"timestamp"`
}

type K8sConfigMapLock struct {
	Clientset *kubernetes.Clientset
	Namespace string
	CMName    string
	Timeout   time.Duration
}

func NewK8sConfigMapLock(namespace, cmName string, timeout time.Duration) (*K8sConfigMapLock, error) {
	config, err := rest.InClusterConfig()
	if err != nil {
		return nil, err
	}
	clientset, err := kubernetes.NewForConfig(config)
	if err != nil {
		return nil, err
	}
	return &K8sConfigMapLock{
		Clientset: clientset,
		Namespace: namespace,
		CMName:    cmName,
		Timeout:   timeout,
	}, nil
}

func (l *K8sConfigMapLock) TryAcquireLock(image, node string) (bool, error) {
	for i := 0; i < 5; i++ {
		cm, err := l.Clientset.CoreV1().ConfigMaps(l.Namespace).Get(context.TODO(), l.CMName, metav1.GetOptions{})
		if err != nil {
			return false, err
		}
		var info K8sLockInfo
		if v, ok := cm.Data["pulling-lock"]; ok && v != "" {
			_ = json.Unmarshal([]byte(v), &info)
			if time.Since(info.Timestamp) < l.Timeout {
				return false, nil // 锁未超时
			}
		}
		// 尝试原子更新
		newInfo := K8sLockInfo{Image: image, Node: node, Timestamp: time.Now()}
		data, _ := json.Marshal(newInfo)
		cm.Data["pulling-lock"] = string(data)
		_, err = l.Clientset.CoreV1().ConfigMaps(l.Namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
		if err == nil {
			return true, nil // 抢锁成功
		}
		// 冲突则重试
		time.Sleep(200 * time.Millisecond)
	}
	return false, nil
}

func (l *K8sConfigMapLock) ReleaseLock(image, node string) error {
	cm, err := l.Clientset.CoreV1().ConfigMaps(l.Namespace).Get(context.TODO(), l.CMName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var info K8sLockInfo
	if v, ok := cm.Data["pulling-lock"]; ok && v != "" {
		_ = json.Unmarshal([]byte(v), &info)
		if info.Node == node && info.Image == image {
			cm.Data["pulling-lock"] = ""
			_, err = l.Clientset.CoreV1().ConfigMaps(l.Namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
			return err
		}
	}
	return nil
}

func (l *K8sConfigMapLock) RefreshLock(image, node string) error {
	cm, err := l.Clientset.CoreV1().ConfigMaps(l.Namespace).Get(context.TODO(), l.CMName, metav1.GetOptions{})
	if err != nil {
		return err
	}
	var info K8sLockInfo
	if v, ok := cm.Data["pulling-lock"]; ok && v != "" {
		_ = json.Unmarshal([]byte(v), &info)
		if info.Node == node && info.Image == image {
			info.Timestamp = time.Now()
			data, _ := json.Marshal(info)
			cm.Data["pulling-lock"] = string(data)
			_, err = l.Clientset.CoreV1().ConfigMaps(l.Namespace).Update(context.TODO(), cm, metav1.UpdateOptions{})
			return err
		}
	}
	return nil
}

func (l *K8sConfigMapLock) GetLockInfo() (*K8sLockInfo, error) {
	cm, err := l.Clientset.CoreV1().ConfigMaps(l.Namespace).Get(context.TODO(), l.CMName, metav1.GetOptions{})
	if err != nil {
		return nil, err
	}
	var info K8sLockInfo
	if v, ok := cm.Data["pulling-lock"]; ok && v != "" {
		_ = json.Unmarshal([]byte(v), &info)
		return &info, nil
	}
	return nil, nil
}
