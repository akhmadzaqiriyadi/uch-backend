package config

// Build-time variables injected via -ldflags during build
var (
	Version   = "1.0.0"
	GitCommit = "development"
	BuildTime = "unspecified"
)

type VersionInfo struct {
	Version   string `json:"version"`
	GitCommit string `json:"git_commit"`
	BuildTime string `json:"build_time"`
}

func GetVersionInfo() VersionInfo {
	return VersionInfo{
		Version:   Version,
		GitCommit: GitCommit,
		BuildTime: BuildTime,
	}
}
