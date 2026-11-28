package moonlighter

import (
	"errors"
	"fmt"
)

// WSLConf returns the /etc/wsl.conf for the moonlighter-managed distro: systemd on with a
// shared mount root, Windows drive automount off, and Windows interop off.
func WSLConf() string {
	return `[boot]
systemd=true
command=mount --make-rshared /

[automount]
enabled=false

[interop]
enabled=false
appendWindowsPath=false
`
}

// PortReserveUnit returns the systemd unit that holds UDP 8472 so mirrored mode keeps carrying
// flannel traffic: wait for the port, bind once, succeed only on EADDRINUSE; PartOf reruns it.
func PortReserveUnit() string {
	return `[Unit]
Description=Reserve UDP 8472 for flannel under WSL mirrored networking
After=k3s-agent.service
PartOf=k3s-agent.service

[Service]
Type=oneshot
RemainAfterExit=yes
TimeoutStartSec=300
ExecStart=/bin/sh -c 'until ss -H -uln "sport = :8472" | grep -q .; do sleep 1; done'
ExecStart=/usr/bin/python3 -c "import errno,socket\ns=socket.socket(socket.AF_INET,socket.SOCK_DGRAM)\ntry:\n    s.bind(('0.0.0.0',8472))\nexcept OSError as e:\n    raise SystemExit(0 if e.errno == errno.EADDRINUSE else 1)\nraise SystemExit(1)"

[Install]
WantedBy=k3s-agent.service
`
}

// Size is a slice of a machine: CPUs and memory in MiB.
type Size struct {
	CPUs   int
	MemMiB int64
}

// AgentArgs returns the k3s agent arguments that join the distro to the cluster at server from
// nodeIP, reserving for the system whatever the distro has beyond size. It takes no join token:
// how the token reaches k3s belongs to the install step.
func AgentArgs(server, nodeIP string, distro, size Size) ([]string, error) {
	switch {
	case server == "":
		return nil, errors.New("empty server URL")
	case nodeIP == "":
		return nil, errors.New("empty node address")
	case size.CPUs <= 0:
		return nil, fmt.Errorf("lent size must have CPUs, got %d", size.CPUs)
	case size.MemMiB <= 0:
		return nil, fmt.Errorf("lent size must have memory, got %d MiB", size.MemMiB)
	case size.CPUs > distro.CPUs:
		return nil, fmt.Errorf("lent size wants %d CPUs of the distro's %d", size.CPUs, distro.CPUs)
	case size.MemMiB > distro.MemMiB:
		return nil, fmt.Errorf("lent size wants %d MiB of the distro's %d MiB", size.MemMiB, distro.MemMiB)
	}
	return []string{
		"agent",
		"--server", server,
		"--node-ip", nodeIP,
		fmt.Sprintf("--kubelet-arg=system-reserved=cpu=%dm,memory=%dMi",
			(distro.CPUs-size.CPUs)*1000, distro.MemMiB-size.MemMiB),
	}, nil
}
