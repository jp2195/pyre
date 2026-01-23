package main

import (
	"context"
	"fmt"
	"os"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/joshuamontgomery/pyre/internal/auth"
	"github.com/joshuamontgomery/pyre/internal/config"
	"github.com/joshuamontgomery/pyre/internal/ssh"
	"github.com/joshuamontgomery/pyre/internal/testutil"
	"github.com/joshuamontgomery/pyre/internal/tui"
)

func main() {
	fmt.Println("Starting mock PAN-OS server...")
	mock := testutil.NewMockPANOS()
	defer mock.Close()

	fmt.Printf("Mock API server running at: %s\n", mock.Host())

	// Start mock SSH server
	fmt.Println("Starting mock SSH server...")
	sshServer, err := ssh.NewMockSSHServer()
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error creating mock SSH server: %v\n", err)
		os.Exit(1)
	}
	sshServer.SetDefaultResponses()

	// Add some troubleshooting-specific responses
	setupTroubleshootingResponses(sshServer)

	if err := sshServer.Start(); err != nil {
		fmt.Fprintf(os.Stderr, "Error starting mock SSH server: %v\n", err)
		os.Exit(1)
	}
	defer sshServer.Close()

	fmt.Printf("Mock SSH server running at: %s:%d\n", sshServer.Host(), sshServer.Port())
	fmt.Println("Login credentials: admin / admin")
	fmt.Println("Starting pyre TUI...")
	fmt.Println("Press '4' or 't' to access Troubleshooting view")

	cfg := config.DefaultConfig()

	// Pre-configure with mock server and generate API key
	creds := &auth.Credentials{
		Host:     mock.Host(),
		APIKey:   "LUFRPT1234567890abcdef==",
		Insecure: true,
	}

	model := tui.NewModel(cfg, creds)

	// Set up SSH connection for the demo
	if err := setupDemoSSH(&model, sshServer); err != nil {
		fmt.Fprintf(os.Stderr, "Warning: Could not set up SSH for demo: %v\n", err)
		fmt.Println("Troubleshooting runbooks requiring SSH will not be available")
	}

	p := tea.NewProgram(
		model,
		tea.WithAltScreen(),
		tea.WithMouseCellMotion(),
	)

	if _, err := p.Run(); err != nil {
		fmt.Fprintf(os.Stderr, "Error running pyre: %v\n", err)
		os.Exit(1)
	}
}

// setupDemoSSH creates and connects an SSH client to the mock server
func setupDemoSSH(model *tui.Model, sshServer *ssh.MockSSHServer) error {
	// Create SSH config for the mock server
	sshCfg := config.SSHConfig{
		Port:     sshServer.Port(),
		Username: "admin",
		Password: "admin",
		Timeout:  10,
	}

	// Create SSH client
	sshClient, err := ssh.NewClient(sshServer.Host(), sshCfg)
	if err != nil {
		return fmt.Errorf("failed to create SSH client: %w", err)
	}

	// Connect to the mock SSH server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sshClient.Connect(ctx); err != nil {
		return fmt.Errorf("failed to connect SSH: %w", err)
	}

	// Configure the TUI model with the SSH client
	model.SetupDemoSSH(sshClient)

	return nil
}

// setupTroubleshootingResponses adds mock responses for troubleshooting runbooks
func setupTroubleshootingResponses(server *ssh.MockSSHServer) {
	// Panorama connectivity responses
	server.SetResponse("show panorama-status", `Panorama Server 1 : 10.0.0.100
    Connected     : yes
    HA state      : primary

Panorama Server 2 : 10.0.0.101
    Connected     : yes
    HA state      : secondary
`)

	// MS log with some warnings for demo
	server.SetResponse("less mp-log ms.log.*", `Jan 21 10:30:00 pan_comm[1234]: Connected to Panorama 10.0.0.100
Jan 21 10:30:01 pan_comm[1234]: SSL handshake successful
Jan 21 10:30:02 pan_comm[1234]: Registration confirmed
Jan 21 10:35:00 pan_comm[1234]: Heartbeat received
Jan 21 10:40:00 pan_comm[1234]: Heartbeat received
Jan 21 10:45:00 pan_comm[1234]: Config push received
Jan 21 10:45:01 pan_comm[1234]: Config applied successfully
`)

	// HA state - active/passive
	server.SetResponse("show high-availability state", `Group 1:
  Mode: Active-Passive
  Local Information:
    Version: 1
    State: active
    Priority: 100
    Preemptive: no

  Peer Information:
    Connection status: up
    Version: 1
    State: passive
    Priority: 100
    Preemptive: no
`)

	// HA state synchronization
	server.SetResponse("show high-availability state-synchronization", `Synchronization:
  Enabled: yes
  Running config: synchronized
  State: synchronized
  Last sync: Jan 21 10:45:00 2026
`)

	// HA link monitoring
	server.SetResponse("show high-availability link-monitoring", `Link Monitoring:
  Enabled: yes
  Failure condition: any

Link Groups:
  Group: ethernet-links
    Enabled: yes
    Failure condition: all
    Links:
      ethernet1/1: up
      ethernet1/2: up
`)

	// Jobs/commit history
	server.SetResponse("show jobs all", `JobID  Type          Status    Result    Description
------------------------------------------------------------
1245   Commit        FIN       OK        Configuration committed
1244   Commit        FIN       OK        Configuration committed
1243   AutoCommit    FIN       OK        Auto-commit
1242   Commit        FIN       OK        Configuration committed
1241   Download      FIN       OK        Content update
`)

	// Config lock status
	server.SetResponse("show config-lock all", `Configuration locks:
  No locks currently held.
`)

	// Config diff
	server.SetResponse("show config diff", `No differences found.
`)

	// Configd log
	server.SetResponse("less mp-log configd.log.*", `Jan 21 10:45:00 configd[5678]: Commit job 1245 started
Jan 21 10:45:01 configd[5678]: Validating configuration
Jan 21 10:45:02 configd[5678]: Configuration validated successfully
Jan 21 10:45:03 configd[5678]: Applying configuration
Jan 21 10:45:05 configd[5678]: Configuration applied
Jan 21 10:45:05 configd[5678]: Commit job 1245 completed successfully
`)

	// License info
	server.SetResponse("request license info", `License entry:
    Feature: PA-VM
    Description: Palo Alto VM-Series
    Serial: 007200001234
    Issued: January 01, 2024
    Expires: January 01, 2027
    Expired: no

License entry:
    Feature: Threat Prevention
    Description: Threat Prevention
    Serial: 007200001234
    Issued: January 01, 2024
    Expires: January 01, 2027
    Expired: no

License entry:
    Feature: WildFire
    Description: WildFire License
    Serial: 007200001234
    Issued: January 01, 2024
    Expires: January 01, 2027
    Expired: no
`)

	// Support status
	server.SetResponse("show system info.*support", `support: active
`)

	// System resources
	server.SetResponse("show system resources", `top - 10:45:00 up 15 days, 3:45,  0 users,  load average: 0.45, 0.52, 0.48
Tasks: 150 total,   1 running, 149 sleeping,   0 stopped,   0 zombie
%Cpu(s):  5.2 us,  2.1 sy,  0.0 ni, 92.1 id,  0.3 wa,  0.0 hi,  0.3 si,  0.0 st
KiB Mem:  16384000 total, 12288000 used,  4096000 free,   256000 buffers

  PID USER      PR  NI    VIRT    RES    SHR S  %CPU %MEM     TIME+ COMMAND
 1234 root      20   0  512000 256000  64000 S   3.2  1.6   5:32.10 mgmtsrvr
 1235 root      20   0  256000 128000  32000 S   1.5  0.8   2:15.30 pan_comm
 1236 root      20   0  128000  64000  16000 S   0.8  0.4   1:05.20 devsrvr
 1237 root      20   0  128000  64000  16000 S   0.5  0.4   0:45.10 logrcvr
`)

	// Session info
	server.SetResponse("show session info", `Session Info:
--------------------------------------------------------------------------------
Number of sessions supported:                   262144
Number of active sessions:                      15432
Number of active TCP sessions:                  12500
Number of active UDP sessions:                  2800
Number of active ICMP sessions:                 132
Number of active BCAST sessions:                0
Number of active MCAST sessions:                0
Session utilization:                            5%

Number of sessions created:                     1543289
Packet rate:                                    25000/s
Throughput:                                     524288 kbps
New connection rate:                            1250 cps
`)

	// Running resource monitor
	server.SetResponse("show running resource-monitor", `Resource utilization (sobel):
--------------------------------------------------------------------------------
Sobel
  CPU utilization:          22%
  Packet rate:              25000 pps
  Session utilization:      5%

  Packet buffer utilization: 8%
  Descriptor utilization:    5%
`)

	// MP logs for OOM check
	server.SetResponse("less mp-log messages.*oom.*", ``)

	// Ping tests
	server.SetResponse("ping host panorama.*", `PING 10.0.0.100 (10.0.0.100) 56(84) bytes of data.
64 bytes from 10.0.0.100: icmp_seq=1 ttl=64 time=0.5 ms
64 bytes from 10.0.0.100: icmp_seq=2 ttl=64 time=0.4 ms
64 bytes from 10.0.0.100: icmp_seq=3 ttl=64 time=0.4 ms

--- 10.0.0.100 ping statistics ---
3 packets transmitted, 3 received, 0.0% packet loss, time 2003ms
rtt min/avg/max/mdev = 0.400/0.433/0.500/0.047 ms
`)

	server.SetResponse("ping host updates.paloaltonetworks.com.*", `PING updates.paloaltonetworks.com (199.167.52.42) 56(84) bytes of data.
64 bytes from 199.167.52.42: icmp_seq=1 ttl=52 time=15.2 ms
64 bytes from 199.167.52.42: icmp_seq=2 ttl=52 time=14.8 ms
64 bytes from 199.167.52.42: icmp_seq=3 ttl=52 time=15.1 ms

--- updates.paloaltonetworks.com ping statistics ---
3 packets transmitted, 3 received, 0.0% packet loss, time 2003ms
rtt min/avg/max/mdev = 14.800/15.033/15.200/0.170 ms
`)

	// HA logs
	server.SetResponse("less mp-log ha_agent.log.*", `Jan 21 10:30:00 ha_agent[2345]: HA heartbeat sent
Jan 21 10:30:01 ha_agent[2345]: HA heartbeat received from peer
Jan 21 10:30:05 ha_agent[2345]: HA state: active
Jan 21 10:30:05 ha_agent[2345]: Peer state: passive
Jan 21 10:35:00 ha_agent[2345]: HA heartbeat sent
Jan 21 10:35:01 ha_agent[2345]: HA heartbeat received from peer
`)

	// Threat prevention license check
	server.SetResponse("request license info.*threat", `License entry:
    Feature: Threat Prevention
    Description: Threat Prevention
    Serial: 007200001234
    Issued: January 01, 2024
    Expires: January 01, 2027
    Expired: no
`)
}
