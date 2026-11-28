package main

import (
	"os/exec"
	"strings"
	"testing"
)

func runMoonlighter(t *testing.T, args ...string) (int, string) {
	t.Helper()
	bin := t.TempDir() + "/moonlighter"
	if out, err := exec.Command("go", "build", "-o", bin, ".").CombinedOutput(); err != nil {
		t.Fatalf("go build: %v: %s", err, out)
	}
	out, err := exec.Command(bin, args...).CombinedOutput()
	code := 0
	if ee, ok := err.(*exec.ExitError); ok {
		code = ee.ExitCode()
	} else if err != nil {
		t.Fatalf("%s: %v", args, err)
	}
	return code, string(out)
}

func TestCheckFailsClosedOnLinux(t *testing.T) {
	code, out := runMoonlighter(t, "check")
	if code != 1 {
		t.Fatalf("moonlighter check exited %d, want 1: the linux signal read fails closed", code)
	}
	if !strings.Contains(out, "error:") {
		t.Fatalf("output %q lacks the error line", out)
	}
	if !strings.Contains(out, "idle:") || !strings.Contains(out, "game:") || !strings.Contains(out, "free_ram:") {
		t.Fatalf("output %q lacks a signal line", out)
	}
}

func TestCheckRejectsOtherArguments(t *testing.T) {
	for _, args := range [][]string{{}, {"check", "extra"}, {"run"}} {
		code, out := runMoonlighter(t, args...)
		if code != 2 || !strings.Contains(out, "usage: moonlighter check") {
			t.Fatalf("moonlighter %v exited %d with output %q, want exit 2 with the usage line", args, code, out)
		}
	}
}
