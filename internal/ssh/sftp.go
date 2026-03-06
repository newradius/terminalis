package ssh

import (
	"fmt"
	"io"
	"os"
	"path"
	"strings"
	"sync"

	"github.com/pkg/sftp"
	gossh "golang.org/x/crypto/ssh"
)

// FileEntry represents a file/directory for the frontend.
type FileEntry struct {
	Name    string `json:"name"`
	Size    int64  `json:"size"`
	IsDir   bool   `json:"isDir"`
	ModTime int64  `json:"modTime"` // unix timestamp
	Mode    string `json:"mode"`    // e.g. "-rwxr-xr-x"
}

// SftpSession wraps an SFTP client for a single SSH connection.
type SftpSession struct {
	mu     sync.Mutex
	client *sftp.Client
	conn   *gossh.Client
}

// NewSftpSession creates an SFTP session on the given SSH connection.
func NewSftpSession(conn *gossh.Client) (*SftpSession, error) {
	if conn == nil {
		return nil, fmt.Errorf("no SSH connection")
	}
	c, err := sftp.NewClient(conn)
	if err != nil {
		return nil, fmt.Errorf("failed to create SFTP client: %w", err)
	}
	return &SftpSession{client: c, conn: conn}, nil
}

// ListDir returns the contents of a remote directory.
func (s *SftpSession) ListDir(dirPath string) ([]FileEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return nil, fmt.Errorf("SFTP session closed")
	}

	entries, err := s.client.ReadDir(dirPath)
	if err != nil {
		return nil, err
	}

	result := make([]FileEntry, 0, len(entries))
	for _, e := range entries {
		result = append(result, FileEntry{
			Name:    e.Name(),
			Size:    e.Size(),
			IsDir:   e.IsDir(),
			ModTime: e.ModTime().Unix(),
			Mode:    e.Mode().String(),
		})
	}
	return result, nil
}

// ReadFileToLocal downloads a remote file to a local path.
func (s *SftpSession) ReadFileToLocal(remotePath, localPath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return fmt.Errorf("SFTP session closed")
	}

	remote, err := s.client.Open(remotePath)
	if err != nil {
		return err
	}
	defer remote.Close()

	local, err := os.Create(localPath)
	if err != nil {
		return err
	}
	defer local.Close()

	_, err = io.Copy(local, remote)
	return err
}

// UploadFile uploads a local file to a remote path.
func (s *SftpSession) UploadFile(localPath, remotePath string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return fmt.Errorf("SFTP session closed")
	}

	local, err := os.Open(localPath)
	if err != nil {
		return err
	}
	defer local.Close()

	remote, err := s.client.Create(remotePath)
	if err != nil {
		return err
	}
	defer remote.Close()

	_, err = io.Copy(remote, local)
	return err
}

// Stat returns file info for a remote path.
func (s *SftpSession) Stat(remotePath string) (*FileEntry, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return nil, fmt.Errorf("SFTP session closed")
	}

	info, err := s.client.Stat(remotePath)
	if err != nil {
		return nil, err
	}

	return &FileEntry{
		Name:    info.Name(),
		Size:    info.Size(),
		IsDir:   info.IsDir(),
		ModTime: info.ModTime().Unix(),
		Mode:    info.Mode().String(),
	}, nil
}

// GetHome returns the user's home directory via SFTP.
func (s *SftpSession) GetHome() string {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client == nil {
		return "/"
	}
	cwd, err := s.client.Getwd()
	if err != nil {
		return "/"
	}
	return cwd
}

// ResolvePath resolves a path (handles relative paths).
func ResolvePath(base, target string) string {
	if strings.HasPrefix(target, "/") {
		return path.Clean(target)
	}
	return path.Clean(path.Join(base, target))
}

// Close closes the SFTP session.
func (s *SftpSession) Close() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.client != nil {
		err := s.client.Close()
		s.client = nil
		return err
	}
	return nil
}

// ExecPwd runs `pwd` on the SSH connection via a new session to get the shell's cwd.
func ExecPwd(conn *gossh.Client) (string, error) {
	if conn == nil {
		return "", fmt.Errorf("no SSH connection")
	}
	session, err := conn.NewSession()
	if err != nil {
		return "", err
	}
	defer session.Close()

	out, err := session.Output("pwd")
	if err != nil {
		return "", err
	}
	return strings.TrimSpace(string(out)), nil
}
