package ssh

import (
	"crypto/rand"
	"crypto/rsa"
	"fmt"
	"io"
	"net"
	"regexp"
	"sync"

	"golang.org/x/crypto/ssh"
)

// MockSSHServer is a mock SSH server for testing purposes.
type MockSSHServer struct {
	mu        sync.RWMutex
	listener  net.Listener
	responses map[string]string // regex pattern -> response
	config    *ssh.ServerConfig
	done      chan struct{}
	host      string
	port      int
}

// NewMockSSHServer creates a new mock SSH server.
func NewMockSSHServer() (*MockSSHServer, error) {
	// Generate a temporary RSA key for the server
	privateKey, err := rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	signer, err := ssh.NewSignerFromKey(privateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to create signer: %w", err)
	}

	config := &ssh.ServerConfig{
		NoClientAuth: true,
		PasswordCallback: func(c ssh.ConnMetadata, pass []byte) (*ssh.Permissions, error) {
			return nil, nil
		},
	}
	config.AddHostKey(signer)

	return &MockSSHServer{
		responses: make(map[string]string),
		config:    config,
		done:      make(chan struct{}),
	}, nil
}

// SetResponse sets a mock response for a command pattern (regex).
func (s *MockSSHServer) SetResponse(pattern, response string) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.responses[pattern] = response
}

// Start starts the mock SSH server on a random available port.
func (s *MockSSHServer) Start() error {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		return fmt.Errorf("failed to listen: %w", err)
	}

	s.listener = listener
	addr := listener.Addr().(*net.TCPAddr)
	s.host = addr.IP.String()
	s.port = addr.Port

	go s.acceptLoop()

	return nil
}

// Close stops the mock SSH server.
func (s *MockSSHServer) Close() error {
	close(s.done)
	if s.listener != nil {
		return s.listener.Close()
	}
	return nil
}

// Host returns the server's host address.
func (s *MockSSHServer) Host() string {
	return s.host
}

// Port returns the server's port.
func (s *MockSSHServer) Port() int {
	return s.port
}

// Address returns the full server address as host:port.
func (s *MockSSHServer) Address() string {
	return fmt.Sprintf("%s:%d", s.host, s.port)
}

func (s *MockSSHServer) acceptLoop() {
	for {
		conn, err := s.listener.Accept()
		if err != nil {
			select {
			case <-s.done:
				return
			default:
				continue
			}
		}

		go s.handleConnection(conn)
	}
}

func (s *MockSSHServer) handleConnection(netConn net.Conn) {
	defer netConn.Close()

	sshConn, chans, reqs, err := ssh.NewServerConn(netConn, s.config)
	if err != nil {
		return
	}
	defer sshConn.Close()

	// Discard out-of-band requests
	go ssh.DiscardRequests(reqs)

	for newChannel := range chans {
		if newChannel.ChannelType() != "session" {
			newChannel.Reject(ssh.UnknownChannelType, "unknown channel type")
			continue
		}

		channel, requests, err := newChannel.Accept()
		if err != nil {
			continue
		}

		go s.handleChannel(channel, requests)
	}
}

func (s *MockSSHServer) handleChannel(channel ssh.Channel, requests <-chan *ssh.Request) {
	defer channel.Close()

	for req := range requests {
		switch req.Type {
		case "exec":
			if len(req.Payload) < 4 {
				req.Reply(false, nil)
				continue
			}

			// Extract command from payload (length-prefixed string)
			cmdLen := int(req.Payload[0])<<24 | int(req.Payload[1])<<16 | int(req.Payload[2])<<8 | int(req.Payload[3])
			if len(req.Payload) < 4+cmdLen {
				req.Reply(false, nil)
				continue
			}
			cmd := string(req.Payload[4 : 4+cmdLen])

			req.Reply(true, nil)

			response := s.getResponse(cmd)
			io.WriteString(channel, response)

			// Send exit status
			channel.SendRequest("exit-status", false, []byte{0, 0, 0, 0})

			// Close stdout to signal command completion
			channel.CloseWrite()
			return

		case "shell":
			req.Reply(true, nil)
			// For shell requests, just close the channel
			return

		default:
			req.Reply(false, nil)
		}
	}
}

func (s *MockSSHServer) getResponse(cmd string) string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	for pattern, response := range s.responses {
		matched, err := regexp.MatchString(pattern, cmd)
		if err == nil && matched {
			return response
		}
	}

	return fmt.Sprintf("Unknown command: %s\n", cmd)
}

// SetDefaultResponses sets up common PAN-OS command responses.
func (s *MockSSHServer) SetDefaultResponses() {
	s.SetResponse("show clock", "Sat Jan 20 10:30:45 PST 2026\n")
	s.SetResponse("show panorama-status", `Panorama Server 1 : 10.0.0.100
    Connected     : yes
    HA state      : primary
`)
	s.SetResponse("show high-availability state", `Group 1:
  Local Information:
    State: active
    Priority: 100
  Peer Information:
    State: passive
    Priority: 100
`)
	s.SetResponse("show high-availability link-monitoring", `Link Monitoring:
  Enabled: yes
  Failure condition: any
  Link Groups:
    ethernet1/1: up
    ethernet1/2: up
`)
	s.SetResponse("show jobs all", `JobId   Type         Status  Result
1234    Commit       FIN     OK
1235    Commit       FIN     FAIL
`)
	s.SetResponse("show config-lock all", "Config lock is not set\n")
	s.SetResponse("request license info", `License entry:
  Feature: PA-VM
  Expires: 2027-01-01
  Expired: no
`)
	s.SetResponse("show system resources", `System resources:
  CPU %: 15.2
  Memory %: 42.3
  Top processes:
    mgmtsrvr: 5.1%
    pan_comm: 3.2%
`)
	s.SetResponse("show session info", `Session Info:
  num-active: 12345
  num-max: 65535
  session-utilization: 18%
`)
	s.SetResponse("show running resource-monitor", `Resource utilization:
  Dataplane CPU: 22%
  Packet rate: 150000 pps
`)
	s.SetResponse("ping host.*", `PING 10.0.0.1 (10.0.0.1): 56 data bytes
64 bytes from 10.0.0.1: icmp_seq=0 ttl=64 time=0.5 ms
64 bytes from 10.0.0.1: icmp_seq=1 ttl=64 time=0.4 ms
64 bytes from 10.0.0.1: icmp_seq=2 ttl=64 time=0.4 ms
--- 10.0.0.1 ping statistics ---
3 packets transmitted, 3 packets received, 0.0% packet loss
`)
	s.SetResponse("less mp-log ms.log.*", `Jan 20 10:30:00 pan_comm: Connected to Panorama
Jan 20 10:30:01 pan_comm: SSL handshake successful
`)
}
