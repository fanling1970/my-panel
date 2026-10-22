//go:build windows

package service

// Detect Windows 下不支持宿主机直通终端
func (s *HostShellService) Detect() HostShellMode {
	return HostShellMode{Available: false, Reason: "当前平台不支持宿主机直通终端"}
}
