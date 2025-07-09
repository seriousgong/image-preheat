package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

var (
	// 回源相关
	RegistryPullTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: RegistryPullTotalName,
			Help: RegistryPullTotalHelp,
		},
		[]string{LabelImage, LabelResult}, // result: success/failed
	)
	RegistryPullDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    RegistryPullDurationName,
			Help:    RegistryPullDurationHelp,
			Buckets: prometheus.ExponentialBuckets(1, 2, 10),
		},
		[]string{LabelImage},
	)

	// P2P 分发相关
	P2PFetchTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: P2PFetchTotalName,
			Help: P2PFetchTotalHelp,
		},
		[]string{LabelImage, LabelPeer},
	)
	P2PFetchDuration = prometheus.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    P2PFetchDurationName,
			Help:    P2PFetchDurationHelp,
			Buckets: prometheus.ExponentialBuckets(0.5, 2, 10),
		},
		[]string{LabelImage, LabelPeer},
	)
	P2PFetchFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: P2PFetchFailedTotalName,
			Help: P2PFetchFailedTotalHelp,
		},
		[]string{LabelImage, LabelPeer, LabelReason},
	)

	// 预热任务相关
	ImagePreheatTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: ImagePreheatTotalName,
			Help: ImagePreheatTotalHelp,
		},
		[]string{LabelImage, LabelSource}, // source: p2p/registry
	)
	ImagePreheatFailedTotal = prometheus.NewCounterVec(
		prometheus.CounterOpts{
			Name: ImagePreheatFailedTotalName,
			Help: ImagePreheatFailedTotalHelp,
		},
		[]string{LabelImage, LabelSource},
	)

	// 当前正在回源拉取的镜像
	RegistryPullingGauge = prometheus.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: RegistryPullingGaugeName,
			Help: RegistryPullingGaugeHelp,
		},
		[]string{LabelImage, LabelNode},
	)
)

func InitMetrics() {
	prometheus.MustRegister(
		RegistryPullTotal,
		RegistryPullDuration,
		P2PFetchTotal,
		P2PFetchDuration,
		P2PFetchFailedTotal,
		ImagePreheatTotal,
		ImagePreheatFailedTotal,
		RegistryPullingGauge,
	)
}
