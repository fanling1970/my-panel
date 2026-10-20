package exec

// hostProcess 宿主机终端进程抽象
type hostProcess interface {
	Read(p []byte) (int, error)
	Write(p []byte) (int, error)
	Resize(cols, rows uint16) error
	Close() error
	// LocalEcho 为 true 时由服务端回显输入（无 PTY 的回退实现）
	LocalEcho() bool
}
