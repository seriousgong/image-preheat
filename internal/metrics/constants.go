package metrics

const (
	// Counter/Histogram/Gauge 名称
	RegistryPullTotalName       = "registry_pull_total"
	RegistryPullDurationName    = "registry_pull_duration_seconds"
	P2PFetchTotalName           = "p2p_fetch_total"
	P2PFetchDurationName        = "p2p_fetch_duration_seconds"
	P2PFetchFailedTotalName     = "p2p_fetch_failed_total"
	ImagePreheatTotalName       = "image_preheat_total"
	ImagePreheatFailedTotalName = "image_preheat_failed_total"
	RegistryPullingGaugeName    = "registry_pulling"

	// 帮助信息
	RegistryPullTotalHelp       = "Total number of registry pulls"
	RegistryPullDurationHelp    = "Duration of registry pulls"
	P2PFetchTotalHelp           = "Total number of successful P2P fetches"
	P2PFetchDurationHelp        = "Duration of P2P fetches"
	P2PFetchFailedTotalHelp     = "Total number of failed P2P fetches"
	ImagePreheatTotalHelp       = "Total number of image preheat tasks"
	ImagePreheatFailedTotalHelp = "Total number of failed image preheat tasks"
	RegistryPullingGaugeHelp    = "Current images being pulled from registry (value=1 means pulling, 0 means not pulling)"

	// label keys
	LabelImage  = "image"
	LabelResult = "result"
	LabelPeer   = "peer"
	LabelReason = "reason"
	LabelSource = "source"
	LabelNode   = "node"

	// 业务相关常量
	SourceP2P       = "p2p"
	SourceRegistry  = "registry"
	ResultSuccess   = "success"
	ResultFailed    = "failed"
	ReasonNetwork   = "network"
	ReasonLoadError = "load_error"
	ReasonHTTPError = "http_error"
)
