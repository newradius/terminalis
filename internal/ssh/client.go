package ssh

import (
	"fmt"
	"io"
	"net"
	"os"
	"sync"
	"time"

	gossh "golang.org/x/crypto/ssh"
)

type ConnectConfig struct {
	Host           string
	Port           int
	Username       string
	Password       string
	PrivateKeyPath string
	Passphrase     string
	UseKey         bool
	Compression    bool
	KeepAlive      int // seconds, 0 = disabled
	Timeout        int // connection timeout seconds
	Cols           int
	Rows           int
}

type Client struct {
	mu      sync.Mutex
	conn    *gossh.Client
	session *gossh.Session
	stdin   io.WriteCloser
	stdout  io.Reader
	stderr  io.Reader
	done    chan struct{}
}

func Connect(cfg ConnectConfig, hostKeyCallback gossh.HostKeyCallback) (*Client, error) {
	var authMethods []gossh.AuthMethod

	if cfg.UseKey && cfg.PrivateKeyPath != "" {
		keyData, err := os.ReadFile(cfg.PrivateKeyPath)
		if err != nil {
			return nil, fmt.Errorf("failed to read private key: %w", err)
		}

		var signer gossh.Signer
		if cfg.Passphrase != "" {
			signer, err = gossh.ParsePrivateKeyWithPassphrase(keyData, []byte(cfg.Passphrase))
		} else {
			signer, err = gossh.ParsePrivateKey(keyData)
		}
		if err != nil {
			return nil, fmt.Errorf("failed to parse private key: %w", err)
		}
		authMethods = append(authMethods, gossh.PublicKeys(signer))
	}

	if cfg.Password != "" {
		authMethods = append(authMethods, gossh.Password(cfg.Password))
		authMethods = append(authMethods, gossh.KeyboardInteractive(
			func(name, instruction string, questions []string, echos []bool) ([]string, error) {
				answers := make([]string, len(questions))
				for i := range answers {
					answers[i] = cfg.Password
				}
				return answers, nil
			},
		))
	}

	if len(authMethods) == 0 {
		return nil, fmt.Errorf("no authentication method provided")
	}

	timeout := time.Duration(cfg.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second
	}

	sshConfig := &gossh.ClientConfig{
		User:            cfg.Username,
		Auth:            authMethods,
		HostKeyCallback: hostKeyCallback,
		Timeout:         timeout,
	}

	addr := fmt.Sprintf("%s:%d", cfg.Host, cfg.Port)
	conn, err := gossh.Dial("tcp", addr, sshConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to connect: %w", err)
	}

	session, err := conn.NewSession()
	if err != nil {
		conn.Close()
		return nil, fmt.Errorf("failed to create session: %w", err)
	}

	cols := cfg.Cols
	if cols == 0 {
		cols = 120
	}
	rows := cfg.Rows
	if rows == 0 {
		rows = 40
	}

	modes := gossh.TerminalModes{
		gossh.ECHO:          1,
		gossh.TTY_OP_ISPEED: 14400,
		gossh.TTY_OP_OSPEED: 14400,
	}

	if err := session.RequestPty("xterm-256color", rows, cols, modes); err != nil {
		session.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to request PTY: %w", err)
	}

	stdin, err := session.StdinPipe()
	if err != nil {
		session.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to get stdin pipe: %w", err)
	}

	stdout, err := session.StdoutPipe()
	if err != nil {
		session.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to get stdout pipe: %w", err)
	}

	stderr, err := session.StderrPipe()
	if err != nil {
		session.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to get stderr pipe: %w", err)
	}

	if err := session.Shell(); err != nil {
		session.Close()
		conn.Close()
		return nil, fmt.Errorf("failed to start shell: %w", err)
	}

	client := &Client{
		conn:    conn,
		session: session,
		stdin:   stdin,
		stdout:  stdout,
		stderr:  stderr,
		done:    make(chan struct{}),
	}

	// Keep-alive
	if cfg.KeepAlive > 0 {
		go client.keepAlive(time.Duration(cfg.KeepAlive) * time.Second)
	}

	// Wait for session to finish
	go func() {
		session.Wait()
		close(client.done)
	}()

	return client, nil
}

func (c *Client) Write(data []byte) (int, error) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.stdin == nil {
		return 0, fmt.Errorf("connection closed")
	}
	return c.stdin.Write(data)
}

func (c *Client) Read(buf []byte) (int, error) {
	return c.stdout.Read(buf)
}

func (c *Client) ReadStderr(buf []byte) (int, error) {
	return c.stderr.Read(buf)
}

func (c *Client) Resize(cols, rows int) error {
	c.mu.Lock()
	defer c.mu.Unlock()
	if c.session == nil {
		return fmt.Errorf("no session")
	}
	return c.session.WindowChange(rows, cols)
}

func (c *Client) Done() <-chan struct{} {
	return c.done
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.stdin != nil {
		c.stdin.Close()
		c.stdin = nil
	}
	if c.session != nil {
		c.session.Close()
		c.session = nil
	}
	if c.conn != nil {
		c.conn.Close()
		c.conn = nil
	}
	return nil
}

func (c *Client) keepAlive(interval time.Duration) {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.mu.Lock()
			if c.conn != nil {
				_, _, err := c.conn.SendRequest("keepalive@openssh.com", true, nil)
				if err != nil {
					c.mu.Unlock()
					return
				}
			}
			c.mu.Unlock()
		case <-c.done:
			return
		}
	}
}

// QuickConnect parses a connection string like "user@host:port" and connects
func QuickConnect(connStr string, password string, hostKeyCallback gossh.HostKeyCallback) (*Client, ConnectConfig, error) {
	cfg := ConnectConfig{
		Port:     22,
		Username: "root",
		Timeout:  30,
	}

	// Parse user@host:port
	remaining := connStr
	if idx := len(remaining) - 1; idx > 0 {
		if atIdx := indexOf(remaining, '@'); atIdx >= 0 {
			cfg.Username = remaining[:atIdx]
			remaining = remaining[atIdx+1:]
		}
	}

	host, port, err := net.SplitHostPort(remaining)
	if err != nil {
		cfg.Host = remaining
	} else {
		cfg.Host = host
		fmt.Sscanf(port, "%d", &cfg.Port)
	}

	cfg.Password = password

	client, err := Connect(cfg, hostKeyCallback)
	return client, cfg, err
}

func indexOf(s string, c byte) int {
	for i := 0; i < len(s); i++ {
		if s[i] == c {
			return i
		}
	}
	return -1
}
