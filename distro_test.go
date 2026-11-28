package moonlighter

import (
	"slices"
	"strings"
	"testing"
)

// sections reads systemd and ini text into section -> the lines of that section, in order.
func sections(text string) map[string][]string {
	out, cur := map[string][]string{}, ""
	for _, line := range strings.Split(text, "\n") {
		line = strings.TrimSpace(line)
		switch {
		case line == "":
		case strings.HasPrefix(line, "["):
			cur = line
		default:
			out[cur] = append(out[cur], line)
		}
	}
	return out
}

func TestWSLConf(t *testing.T) {
	s := sections(WSLConf())
	want := map[string][]string{
		"[boot]":      {"systemd=true", "command=mount --make-rshared /"},
		"[automount]": {"enabled=false"},
		"[interop]":   {"enabled=false"},
	}
	for sec, lines := range want {
		for _, l := range lines {
			if !slices.Contains(s[sec], l) {
				t.Errorf("%s lacks %q: %q", sec, l, s[sec])
			}
		}
	}
}

func TestPortReserveUnit(t *testing.T) {
	text := PortReserveUnit()
	s := sections(text)
	for sec, l := range map[string]string{
		"[Unit]":    "After=k3s-agent.service",
		"[Install]": "WantedBy=k3s-agent.service",
	} {
		if !slices.Contains(s[sec], l) {
			t.Errorf("%s lacks %q: %q", sec, l, s[sec])
		}
	}
	if !slices.Contains(s["[Unit]"], "PartOf=k3s-agent.service") {
		t.Errorf("[Unit] lacks PartOf: %q", s["[Unit]"])
	}
	if !slices.Contains(s["[Service]"], "Type=oneshot") {
		t.Errorf("[Service] lacks Type=oneshot: %q", s["[Service]"])
	}
	var exec []string
	for _, l := range s["[Service]"] {
		if strings.HasPrefix(l, "ExecStart=") {
			exec = append(exec, l)
		}
	}
	if len(exec) != 2 {
		t.Fatalf("want two ExecStart lines, got %q", exec)
	}
	for _, w := range []string{"ss ", "8472", "sleep"} {
		if !strings.Contains(exec[0], w) {
			t.Errorf("the first ExecStart must wait for the port, lacks %q: %s", w, exec[0])
		}
	}
	for _, w := range []string{".bind(", "8472", "EADDRINUSE"} {
		if !strings.Contains(exec[1], w) {
			t.Errorf("the second ExecStart must make the bind, lacks %q: %s", w, exec[1])
		}
	}
	if n := strings.Count(text, ".bind("); n != 1 {
		t.Errorf("want exactly one bind, got %d", n)
	}
	if strings.ContainsAny(text, "$%") {
		t.Errorf("systemd would expand $ or %%: %s", text)
	}
}

func TestAgentArgs(t *testing.T) {
	distro := Size{CPUs: 8, MemMiB: 16384}
	args, err := AgentArgs("https://roost.example:6443", "10.0.10.180", distro, Size{CPUs: 4, MemMiB: 8192})
	if err != nil {
		t.Fatal(err)
	}
	for _, pair := range [][2]string{{"--server", "https://roost.example:6443"}, {"--node-ip", "10.0.10.180"}} {
		i := slices.Index(args, pair[0])
		if i < 0 || i+1 >= len(args) || args[i+1] != pair[1] {
			t.Errorf("want %s %s in %q", pair[0], pair[1], args)
		}
	}
	if !slices.Contains(args, "--kubelet-arg=system-reserved=cpu=4000m,memory=8192Mi") {
		t.Errorf("system-reserved must be the distro minus the size: %q", args)
	}
	if strings.Contains(strings.Join(args, " "), "flannel-iface") {
		t.Errorf("--flannel-iface must never be passed: %q", args)
	}
	full, err := AgentArgs("https://roost.example:6443", "10.0.10.180", distro, distro)
	if err != nil || !slices.Contains(full, "--kubelet-arg=system-reserved=cpu=0m,memory=0Mi") {
		t.Errorf("size equal to the distro reserves nothing: %q, %v", full, err)
	}
}

func TestAgentArgsRefusals(t *testing.T) {
	distro, ok := Size{CPUs: 8, MemMiB: 16384}, Size{CPUs: 4, MemMiB: 8192}
	cases := []struct {
		name, server, ip string
		size             Size
	}{
		{"empty server", "", "10.0.10.180", ok},
		{"empty node address", "https://roost.example:6443", "", ok},
		{"zero CPUs", "https://roost.example:6443", "10.0.10.180", Size{0, 8192}},
		{"zero memory", "https://roost.example:6443", "10.0.10.180", Size{4, 0}},
		{"more CPUs than the distro", "https://roost.example:6443", "10.0.10.180", Size{9, 8192}},
		{"more memory than the distro", "https://roost.example:6443", "10.0.10.180", Size{4, 16385}},
	}
	for _, c := range cases {
		if args, err := AgentArgs(c.server, c.ip, distro, c.size); err == nil {
			t.Errorf("%s: want an error, got %q", c.name, args)
		}
	}
}
