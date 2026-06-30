package config

const (
	DefaultDataDir = "/var/lib/xnode"
	DefaultLogDir  = "/var/log/xnode"
	DefaultXrayBin = "/usr/local/bin/xray"
)

type LocalConfig struct {
	PanelURL    string
	NodeID      int64
	NodeDomain  string
	DataDir     string
	LogDir      string
	XrayBinPath string
	EnrollToken string
}
