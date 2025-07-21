package preheat

import (
	"encoding/json"
	"fmt"
	"image-preheat/internal/config"
	"net"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/rs/zerolog/log"
)

// PeerSelector 节点选择器
type PeerSelector struct {
	mu         sync.RWMutex
	peers      []string
	index      int
	lastUpdate time.Time
}

// NewPeerSelector 创建节点选择器
func NewPeerSelector() *PeerSelector {
	return &PeerSelector{
		peers:      []string{},
		index:      0,
		lastUpdate: time.Time{},
	}
}

// UpdatePeers 更新节点列表
func (ps *PeerSelector) UpdatePeers() error {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	var peers []string
	var err error

	// 优先用优选接口
	peers, err = discoverPreferredPeers(config.NodeName)
	if err != nil || len(peers) == 0 {
		log.Warn().Err(err).Msg("优选peers接口失败，fallback到headless service")
		peers, err = discoverPeersFromHeadlessService()
		if err != nil {
			return fmt.Errorf("发现 peers 失败: %v", err)
		}
	}

	ps.peers = peers
	ps.lastUpdate = time.Now()
	log.Info().Strs("peers", peers).Msg("更新 peers 列表")
	return nil
}

// GetNextPeer 获取下一个 peer (轮询)
func (ps *PeerSelector) GetNextPeer() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if len(ps.peers) == 0 {
		return ""
	}

	peer := ps.peers[ps.index%len(ps.peers)]
	ps.index++
	return peer
}

// GetRandomPeer 随机获取一个 peer
func (ps *PeerSelector) GetRandomPeer() string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	if len(ps.peers) == 0 {
		return ""
	}

	// 简单的随机选择 (基于时间戳)
	index := int(time.Now().UnixNano()) % len(ps.peers)
	return ps.peers[index]
}

// GetPeers 获取所有 peers
func (ps *PeerSelector) GetPeers() []string {
	ps.mu.RLock()
	defer ps.mu.RUnlock()

	result := make([]string, len(ps.peers))
	copy(result, ps.peers)
	return result
}

// discoverPeersFromHeadlessService 从 headless service 发现 peers
func discoverPeersFromHeadlessService() ([]string, error) {
	// 解析 headless service DNS
	serviceName := config.PeerDiscoveryServiceName
	ips, err := net.LookupHost(serviceName)
	if err != nil {
		return nil, fmt.Errorf("DNS 解析失败: %v", err)
	}

	// 过滤掉自己的 IP
	myIP := getMyPodIP()
	var peers []string
	for _, ip := range ips {
		if ip != myIP {
			peers = append(peers, ip)
		}
	}

	return peers, nil
}

// discoverPreferredPeers 从本地 server 获取优选的 peer IP 列表
func discoverPreferredPeers(nodeName string) ([]string, error) {
	url := "http://" + config.PeersServerName + ":8080/peers/priority?node=" + nodeName
	resp, err := http.Get(url)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("优选peers接口返回非200: %d", resp.StatusCode)
	}
	var peers []string
	if err := json.NewDecoder(resp.Body).Decode(&peers); err != nil {
		return nil, err
	}
	return peers, nil
}

// getMyPodIP 获取当前 Pod 的 IP
func getMyPodIP() string {
	// 方法1: 从环境变量获取 (推荐)
	if podIP := os.Getenv("POD_IP"); podIP != "" {
		return podIP
	}

	// 方法2: 从网络接口获取
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		log.Error().Err(err).Msg("获取网络接口失败")
		return ""
	}

	for _, addr := range addrs {
		if ipnet, ok := addr.(*net.IPNet); ok && !ipnet.IP.IsLoopback() {
			if ipnet.IP.To4() != nil {
				return ipnet.IP.String()
			}
		}
	}

	return ""
}

// 全局 peer 选择器实例
var peerSelector = NewPeerSelector()

// 定期更新 peers 列表
func StartPeerDiscovery() {
	log.Info().Msg("启动节点发现定时任务")
	ticker := time.NewTicker(config.PeerDiscoveryInterval)
	defer ticker.Stop()

	for {
		<-ticker.C
		log.Debug().Msg("周期性刷新 peers 列表")
		if err := peerSelector.UpdatePeers(); err != nil {
			log.Error().Err(err).Msg("更新 peers 失败")
		}
	}
}

// 获取 peers 列表 (供外部使用)
func GetPeerIPs() []string {
	return peerSelector.GetPeers()
}
