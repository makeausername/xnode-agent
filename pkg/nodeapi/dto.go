package nodeapi

type APIResponse[T any] struct {
	Ret       int    `json:"ret"`
	Data      T      `json:"data,omitempty"`
	Msg       string `json:"msg,omitempty"`
	Code      string `json:"code,omitempty"`
	RequestID string `json:"request_id,omitempty"`
}

type NodeConfig struct {
	SchemaVersion int           `json:"schema_version"`
	NodeID        int64         `json:"node_id"`
	Domain        string        `json:"domain"`
	Profile       NodeProfile   `json:"profile"`
	Reality       RealityConfig `json:"reality"`
	Limits        NodeLimits    `json:"limits"`
	Report        ReportConfig  `json:"report"`
	ConfigHash    string        `json:"config_hash,omitempty"`
}

type NodeProfile struct {
	Name     string `json:"name"`
	Protocol string `json:"protocol"`
	Network  string `json:"network"`
	Security string `json:"security"`
	Flow     string `json:"flow"`
	Listen   string `json:"listen"`
	Port     int    `json:"port"`
}

type RealityConfig struct {
	Target      string   `json:"target"`
	ServerNames []string `json:"server_names"`
	Fingerprint string   `json:"fingerprint"`
	PublicKey   string   `json:"public_key,omitempty"`
	ShortIDs    []string `json:"short_ids,omitempty"`
}

type NodeLimits struct {
	MaxUsers int `json:"max_users,omitempty"`
}

type ReportConfig struct {
	UserSyncIntervalSec  int `json:"user_sync_interval_sec"`
	TrafficIntervalSec   int `json:"traffic_interval_sec"`
	OnlineIntervalSec    int `json:"online_interval_sec"`
	HeartbeatIntervalSec int `json:"heartbeat_interval_sec"`
}

type UserInfo struct {
	ID             int64  `json:"id"`
	UUID           string `json:"uuid"`
	Email          string `json:"email"`
	SpeedLimitMbps int    `json:"speed_limit_mbps"`
	IPLimit        int    `json:"ip_limit"`
	Enabled        bool   `json:"enabled"`
	UpdatedAt      int64  `json:"updated_at"`
}

type UserTraffic struct {
	UserID   int64 `json:"user_id"`
	Upload   int64 `json:"u"`
	Download int64 `json:"d"`
}

type DetectRule struct {
	ID      int64  `json:"id"`
	Type    string `json:"type"`
	Pattern string `json:"pattern"`
}

type OnlineIP struct {
	UserID int64  `json:"user_id"`
	IP     string `json:"ip"`
}

type HostInfo struct {
	OS         string `json:"os"`
	Arch       string `json:"arch"`
	Kernel     string `json:"kernel"`
	PublicIPv4 string `json:"public_ipv4,omitempty"`
	PublicIPv6 string `json:"public_ipv6,omitempty"`
}

type EnrollRequest struct {
	NodeID             int64    `json:"node_id"`
	Domain             string   `json:"domain"`
	AgentVersion       string   `json:"agent_version"`
	InstallFingerprint string   `json:"install_fingerprint"`
	Host               HostInfo `json:"host"`
}

type EnrollResponse struct {
	NodeToken         string `json:"node_token"`
	PanelURL          string `json:"panel_url"`
	NodeID            int64  `json:"node_id"`
	Domain            string `json:"domain"`
	ReportIntervalSec int    `json:"report_interval_sec"`
	ConfigIntervalSec int    `json:"config_interval_sec"`
}

type RuntimeReport struct {
	NodeID       int64    `json:"node_id"`
	AgentVersion string   `json:"agent_version"`
	CoreVersion  string   `json:"core_version"`
	State        string   `json:"state"`
	PublicKey    string   `json:"public_key"`
	ShortIDs     []string `json:"short_ids"`
	Capabilities []string `json:"capabilities"`
	ConfigHash   string   `json:"config_hash,omitempty"`
	LastError    string   `json:"last_error,omitempty"`
}

type TrafficReport struct {
	ReportID    string        `json:"report_id"`
	NodeID      int64         `json:"node_id"`
	PeriodStart int64         `json:"period_start"`
	PeriodEnd   int64         `json:"period_end"`
	Data        []UserTraffic `json:"data"`
}

type OnlineReport struct {
	ReportID    string     `json:"report_id"`
	NodeID      int64      `json:"node_id"`
	PeriodStart int64      `json:"period_start"`
	PeriodEnd   int64      `json:"period_end"`
	Data        []OnlineIP `json:"data"`
}

type DetectLogItem struct {
	UserID    int64  `json:"user_id"`
	RuleID    int64  `json:"rule_id"`
	IP        string `json:"ip"`
	Target    string `json:"target"`
	CreatedAt int64  `json:"created_at"`
}

type DetectLogReport struct {
	ReportID    string          `json:"report_id"`
	NodeID      int64           `json:"node_id"`
	PeriodStart int64           `json:"period_start"`
	PeriodEnd   int64           `json:"period_end"`
	Data        []DetectLogItem `json:"data"`
}

type HeartbeatReport struct {
	NodeID       int64  `json:"node_id"`
	AgentVersion string `json:"agent_version"`
	CoreVersion  string `json:"core_version"`
	State        string `json:"state"`
	LastError    string `json:"last_error,omitempty"`
	ConfigHash   string `json:"config_hash,omitempty"`
}
