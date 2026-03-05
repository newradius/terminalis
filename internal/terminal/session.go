package terminal

// TerminalSession is the common interface for both SSH and local shell sessions.
// ssh.Client already satisfies this interface.
type TerminalSession interface {
	Read(buf []byte) (int, error)
	ReadStderr(buf []byte) (int, error)
	Write(data []byte) (int, error)
	Resize(cols, rows int) error
	Done() <-chan struct{}
	Close() error
}
