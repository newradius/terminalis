package ssh

import (
	"bufio"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"net"
	"os"
	"path/filepath"
	"strings"
	"sync"

	gossh "golang.org/x/crypto/ssh"
)

type HostKeyStatus int

const (
	HostKeyOK       HostKeyStatus = iota
	HostKeyNew                    // not seen before
	HostKeyMismatch               // key changed
)

type HostKeyResult struct {
	Status      HostKeyStatus
	Fingerprint string
	KeyType     string
	Host        string
}

type KnownHosts struct {
	mu   sync.Mutex
	file string
	keys map[string]string // host -> fingerprint
}

func NewKnownHosts(dataDir string) (*KnownHosts, error) {
	kh := &KnownHosts{
		file: filepath.Join(dataDir, "known_hosts"),
		keys: make(map[string]string),
	}
	kh.load()
	return kh, nil
}

func (kh *KnownHosts) load() {
	f, err := os.Open(kh.file)
	if err != nil {
		return
	}
	defer f.Close()

	scanner := bufio.NewScanner(f)
	for scanner.Scan() {
		line := strings.TrimSpace(scanner.Text())
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		parts := strings.SplitN(line, " ", 2)
		if len(parts) == 2 {
			kh.keys[parts[0]] = parts[1]
		}
	}
}

func (kh *KnownHosts) Check(host string, port int, key gossh.PublicKey) HostKeyResult {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	addr := normalizeAddr(host, port)
	fp := fingerprint(key)

	result := HostKeyResult{
		Fingerprint: fp,
		KeyType:     key.Type(),
		Host:        addr,
	}

	if stored, ok := kh.keys[addr]; ok {
		if stored == fp {
			result.Status = HostKeyOK
		} else {
			result.Status = HostKeyMismatch
		}
	} else {
		result.Status = HostKeyNew
	}

	return result
}

func (kh *KnownHosts) Accept(host string, port int, key gossh.PublicKey) error {
	kh.mu.Lock()
	defer kh.mu.Unlock()

	addr := normalizeAddr(host, port)
	fp := fingerprint(key)
	kh.keys[addr] = fp

	return kh.save()
}

func (kh *KnownHosts) save() error {
	var lines []string
	for host, fp := range kh.keys {
		lines = append(lines, fmt.Sprintf("%s %s", host, fp))
	}
	return os.WriteFile(kh.file, []byte(strings.Join(lines, "\n")+"\n"), 0600)
}

func normalizeAddr(host string, port int) string {
	if port == 22 {
		return host
	}
	return fmt.Sprintf("[%s]:%d", host, port)
}

func fingerprint(key gossh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return key.Type() + " " + base64.StdEncoding.EncodeToString(hash[:])
}

func FingerprintDisplay(key gossh.PublicKey) string {
	hash := sha256.Sum256(key.Marshal())
	return fmt.Sprintf("SHA256:%s", base64.StdEncoding.EncodeToString(hash[:]))
}

// HostKeyCallback returns a gossh.HostKeyCallback that checks against known hosts
// and calls promptFn when a new or changed host key is encountered.
// promptFn should return true to accept the key.
func (kh *KnownHosts) HostKeyCallback(promptFn func(result HostKeyResult) bool) gossh.HostKeyCallback {
	return func(hostname string, remote net.Addr, key gossh.PublicKey) error {
		host, portStr, _ := net.SplitHostPort(remote.String())
		port := 22
		if portStr != "" {
			fmt.Sscanf(portStr, "%d", &port)
		}

		result := kh.Check(host, port, key)

		switch result.Status {
		case HostKeyOK:
			return nil
		case HostKeyNew:
			if promptFn(result) {
				return kh.Accept(host, port, key)
			}
			return fmt.Errorf("host key rejected by user")
		case HostKeyMismatch:
			if promptFn(result) {
				return kh.Accept(host, port, key)
			}
			return fmt.Errorf("host key mismatch - possible MITM attack, rejected by user")
		}
		return nil
	}
}
