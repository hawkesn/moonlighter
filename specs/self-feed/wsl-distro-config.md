---
template_version: 0.6.2
planner_model: claude-sonnet-5-5
builder_model: pending-run
target_repo: https://github.com/hawkesn/moonlighter.git@START_SHA_PENDING
unit: wsl-distro-config
created: 2026-11-28
---

# Run the agent in a Linux distro that moonlighter manages on Windows: distro config pieces Implementation Plan

**Target Repo:** https://github.com/hawkesn/moonlighter.git@START_SHA_PENDING

## Decisions parked for tier-0

- None: the change adds one new Go file in the repo root. It touches no protected path (the root `CODEOWNERS` protects only `/.github/workflows/`), no credential or auth (the join token is not handled here), no migration and no planner flag.

## Seams

#### (a) Stage table

| Order | Piece | Content | What keeps main green |
|---|---|---|---|
| 1 | preflight check | `Preflight` (R14, R15) | merged; pure code with its own tests |
| 2 | install pure pieces | `MergeWSLConfig` (R3) and `VerifyRootfs` (R2 sha256 check) | merged; pure code with its own tests |
| 3 | Task 1 of this plan | `WSLConf` (R16), `PortReserveUnit` (R17, R18), `AgentArgs` (R4, R10) | new pure code with its own tests; nothing calls it yet |
| 4 | install step | rootfs import, writes of the texts above into the distro, firewall rule (R5), logon task (R6), remote one-shot task (R13), post-join check (R19), run on Windows | later plan; calls every piece above |
| 5 | stop | keep-open `wsl.exe` child (R7), stop order (R8), RAM return (R9) | later plan |

#### (b) Shared contracts

- `Preflight(h Host) error` returns nil, or a `*PreflightError` with `Reason` and `Command`.
- `MergeWSLConfig(config string) string` returns the text with `networkingMode=mirrored` set in `[wsl2]`.
- `VerifyRootfs(r io.Reader, pin string) error` returns nil only when the sha256 of the bytes read equals the pin.
- `WSLConf() string` returns the full text of the distro's `/etc/wsl.conf`.
- `PortReserveUnit() string` returns the full text of the port-reserve systemd unit.
- `AgentArgs(server, nodeIP string, distro, size Size) ([]string, error)` returns the arguments after `k3s`, starting with `agent`.

#### (c) Stitch record

| Piece | Plan file | PR | Merge sha | Deviation |
|---|---|---|---|---|
| preflight check | `specs/self-feed/wsl-preflight-check.md` | | | none |
| install pure pieces | `specs/self-feed/wsl-install-pure-pieces.md` | | | none |
| Task 1 | `specs/self-feed/wsl-distro-config.md` | | | none |

## Intent

#### Goal

Three pure helpers that produce what the install step writes into the moonlighter-managed WSL2 distro: its `/etc/wsl.conf`, the port-reserve unit that makes mirrored mode carry pod traffic, and the `k3s agent` arguments with the node address and the size cap. Later plans write them into the distro and run them on Windows.

#### Operator intent

- I1: "Another thought is that it opens up a virtual machine on the host machine?"
- I2: "it's a simple application (windows or linux or macos), where you install it, and you give it a connection url and a token. It should just join."
- I3: "this program should be able to run on any machine/any device (ie. pi, mac, linux, windows)"

#### Choices made

- Epic service: the distro is the virtual machine of I1, and pasting one connection string must join the node (I2). The texts here are what makes the joined node carry pod traffic and stay inside the lent size, which the Epic's promise (lend the machine only as far as its owner allows) needs. All three helpers take plain data and read nothing from the machine, so they build and test on every OS (I3).
- The change lives in a new `distro.go` in the root package `moonlighter`; the card names no file. `install.go`, `lend.go` and `preflight.go` sit in the same root, so no subpackage.
- Taken: this plan covers only the three text and argument builders (card R4, R10, R16, R17, R18). Alternative: plan the whole install step. Impact: it cannot fit one task of at most 120 changed lines and needs Windows to prove; Seams rows 4 and 5 name the later pieces, so this plan says "relates" and the issue stays open.
- Taken: tests go in a new `distro_test.go`, because there is no `distro` test file yet. Alternative: add to `install_test.go`. Impact: that edits an existing file another plan owns.
- Taken: `AgentArgs` takes no token. The join token and how it reaches k3s (environment or file) belong to the install step and the setup command of hawkesn/embersuite#1950; passing a secret on a command line is a choice that plan makes. Alternative: add a `--token` argument here. Impact: the secret would sit in the process list, and this plan would decide a credential question.
- Taken: `AgentArgs` sets allocatable through `--kubelet-arg=system-reserved=cpu=<n>m,memory=<n>Mi`, where reserved is the distro's size minus the lent size (card R4). Kubelet subtracts system-reserved and its eviction threshold from capacity, so allocatable is at most the size. Alternative: `kube-reserved`. Impact: the card names system-reserved.
- Taken: a size of zero, or larger than the distro, is an error, and an empty server or node address is an error. Alternative: clamp. Impact: a clamp would lend more or less than the owner set without saying so.
- Taken: `/etc/wsl.conf` also sets `appendWindowsPath=false` under `[interop]`; the bridged test set it and it is harmless with interop off. Alternative: leave it out. Impact: none either way.
- Taken: the port-reserve unit is `Type=oneshot` with `RemainAfterExit=yes` and `TimeoutStartSec=300`, and two `ExecStart=` lines: one waits with `ss` until UDP 8472 shows, one makes the single bind and exits 0 only on EADDRINUSE. `PartOf` stops it when the agent stops, so a plain start of the agent runs it again (T7); the timeout stops the wait from running forever when the port never opens. Alternative: repeat every 60 s like the hand install. Impact: the card (R17, R18) says once per agent start.
- Taken: review class `prod-code`; `risk_class` is `ordinary` because no workflow, manifest or auth changes.
- Taken: there is no `package.json`, so `npm ci` does not apply; the proof ran with the Go toolchain. The repo has no `test/integration` directory and no Postgres is needed.

#### Non-goals

- No write to the distro, no call of `wsl.exe`, `systemctl` or `k3s`, no firewall rule, no scheduled task.
- No token handling and no connection-string parsing (hawkesn/embersuite#1950).
- No change to `install.go`, `preflight.go`, `lend.go`, `go.mod` or `.github/`.
- No change to the hand installs.

## Constraints

#### Requirements

- R1: `WSLConf` turns systemd on, sets `[boot]` `command=mount --make-rshared /`, and turns automount and interop off (card R16).
- R2: `PortReserveUnit` has `After=`, `PartOf=` (in `[Unit]`) and `WantedBy=` (in `[Install]`) for `k3s-agent.service`, waits with `ss` for UDP 8472, then makes exactly one bind on it that counts EADDRINUSE as success (card R17, R18).
- R3: `AgentArgs` passes `--server` and `--node-ip`, never `--flannel-iface` (card R10).
- R4: `AgentArgs` sets system-reserved to the distro's CPU and memory minus the size, and refuses an empty server or node address, a size of zero or a size above the distro (card R4).

## Validation

#### Acceptance criteria

- Given the generated `wsl.conf`, when it is read, then `[boot]` holds `systemd=true` and `command=mount --make-rshared /`, `[automount]` holds `enabled=false` and `[interop]` holds `enabled=false`.
- Given the generated unit, when it is read, then it holds `After=`, `PartOf=` and `WantedBy=` for `k3s-agent.service`, and the first `ExecStart=` waits for the port and the second makes the one bind.
- Given a distro of 8 CPUs and 16384 MiB and a size of 4 CPUs and 8192 MiB, when `AgentArgs` runs, then it holds `--kubelet-arg=system-reserved=cpu=4000m,memory=8192Mi` and no `--flannel-iface`.
- Given a size above the distro, or zero, when `AgentArgs` runs, then it returns an error.

#### Edge cases

- Size equal to the distro: reserved is `cpu=0m,memory=0Mi`.
- UDP 8472 never opens: the wait stops at the 300 s start timeout and the unit fails, which the post-join check (card R19) in a later plan shows.
- The bind succeeds: the second `ExecStart=` exits 1 and the unit fails.
- Empty server or node address: an error, no arguments.

#### Tests

The tests live in the new `distro_test.go`. The oracle is `go test doc.go distro.go distro_test.go`. The unit and the arguments are guards, and eight wrong patches under `test/fixtures` (`distro_test-wrong-9` to `-16`) stand in for a mutation probe: automount on, no `command=`, no `PartOf=`, no `WantedBy=`, bind before the wait, `--flannel-iface` added, reserved set to the size, and no size check.

- A1 [R1, T1] wsl-conf-table: systemd on, `/` made shared, automount and interop off.
- A2 [R2, T2, T3] port-reserve-unit: the three k3s-agent lines, two `ExecStart=` lines in order, one bind, no `$` or `%`.
- A3 [R3, R4, T4] agent-args: server, node address, system-reserved from distro minus size, no `--flannel-iface`, size equal to the distro.
- A4 [R4, T4] agent-args-refusals: empty server, empty node address, zero CPUs, zero memory, too many CPUs, too much memory.

## Risk

#### Failure modes

- If the pinned rootfs has no `python3` or `ss`, the unit fails at start and mirrored traffic does not cross nodes; the post-join cross-node check (card R19) in a later plan shows it. This plan cannot read the hand-install scripts, which sit outside the repo, so it cannot check the tools.
- If `ss -uln` does not list the flannel VXLAN socket, the wait ends at the 300 s timeout; the same check shows it.

## Traps / gotchas

- T1: Contract: `WSLConf` returns exactly `[boot]`, `systemd=true`, `command=mount --make-rshared /`, then `[automount]` `enabled=false`, then `[interop]` `enabled=false` and `appendWindowsPath=false`, each key on its own line, ending in a newline. Failing input: `[automount]` with `enabled=true` lets Windows drives in.
- T2: Contract: `PortReserveUnit` has `PartOf=k3s-agent.service` and `After=k3s-agent.service` under `[Unit]` and `WantedBy=k3s-agent.service` under `[Install]`. `PartOf` passes only a stop or restart, so without `WantedBy=` a plain agent start does not run the unit. The first `ExecStart=` runs `ss` and `sleep` until UDP 8472 shows; the second has the only `.bind(`, on port 8472, and exits 0 on `EADDRINUSE`, else 1. Failing input: the two lines swapped binds before flannel opens the port, and WSL drops that reservation after 60 s.
- T3: Trap: systemd expands `$` and `%` in `ExecStart=` words. The unit text holds neither. The python program is one double-quoted word whose line breaks are written `\n` (systemd turns them into newlines); a raw newline would end the word.
- T4: Contract: `AgentArgs` returns `agent`, `--server`, the URL, `--node-ip`, the address, then `--kubelet-arg=system-reserved=cpu=<(distro.CPUs-size.CPUs)*1000>m,memory=<distro.MemMiB-size.MemMiB>Mi`. It never returns `--flannel-iface`, because mirrored interface names move across restarts. Failing input: a size of 9 CPUs on a distro of 8 returns an error, never a negative reserve.

## Hazards

Tables declared: none. Every table is `not-touched`: the change is one Go file with no shell, regex, SQL or doc table to run. The unit text holds shell and python, but only as a string the tests read.

## External calls

- none: the three helpers return text and arguments and call nothing outside the repo; the tools named inside the unit (`ss`, `python3`) run later in the distro, in a later plan.

## Tasks

### Task 1: Add WSLConf, PortReserveUnit and AgentArgs

**Belief:** orion-mk4-gaming-pc-specs

**Intent:** I1, I2, I3

**Goal.** Add `distro.go` in package `moonlighter` with `WSLConf() string`, `PortReserveUnit() string`, the type `Size struct{ CPUs int; MemMiB int64 }` and `AgentArgs(server, nodeIP string, distro, size Size) ([]string, error)`.

**Leave alone.**
- Change no file except `distro.go`.
- Leave `distro_test.go` as it is. It holds the tests you must make pass.
- Do not touch `go.mod`, `install.go`, `preflight.go`, `lend.go` or `.github/`.

**Facts measured at the start commit.**
- `ls *.go` shows `doc.go`, `install.go`, `lend.go`, `preflight.go` and their tests; `go.mod` says `go 1.26.10`.
- `grep -n "func " install.go preflight.go lend.go` shows no `WSLConf`, `PortReserveUnit` or `AgentArgs`, and `grep -rn "type Size" *.go` finds nothing in the root package.
- There is no `test/integration` directory.

**The failing test.** `go test doc.go distro.go distro_test.go` fails: `distro.go` does not exist, so `WSLConf`, `PortReserveUnit`, `Size` and `AgentArgs` are undefined.

**The change.** `WSLConf` returns the text in T1. `PortReserveUnit` returns a unit with `[Unit]` (`Description=`, `After=k3s-agent.service`, `PartOf=k3s-agent.service`), `[Service]` (`Type=oneshot`, `RemainAfterExit=yes`, `TimeoutStartSec=300`, then the two `ExecStart=` lines of T2) and `[Install]` (`WantedBy=k3s-agent.service`). The first `ExecStart=` is `/bin/sh -c 'until ss -H -uln "sport = :8472" | grep -q .; do sleep 1; done'`. The second is `/usr/bin/python3 -c "..."` with a program that opens an IPv4 UDP socket, binds `('0.0.0.0', 8472)` in a `try`, exits 0 when the `OSError` has `errno == errno.EADDRINUSE` and 1 for any other error, and exits 1 if the bind succeeds; write its line breaks as `\n` and use single quotes inside. `AgentArgs` returns the errors and the slice of T4 and nothing else.

**Done when.** `go test doc.go distro.go distro_test.go` exits 0, and `gofmt -l .` and `go vet ./...` print nothing.

**Task Contract:**

- item: wsl-distro-config
- relates: hawkesn/embersuite#1955
- rung: 1
- risk_class: ordinary
- repo: https://github.com/hawkesn/moonlighter.git
- base_sha: START_SHA_PENDING
- branch: autoloop/wsl-distro-config
- task_class: build-to-green-against-existing-tests
- oracle_cmd: `go test doc.go distro.go distro_test.go`
- test_cmd: `go test doc.go preflight.go preflight_test.go preflight_units_test.go`
- max_added: 75
- max_deleted: 0
- test_files:
  - "distro_test.go"
- allowed_files:
  - "distro.go"
- release: batch

## RED start files

#### test edit

- bytes: sha256:470e1cf69243acc39f3a2b0bc2b61e378fa8c32c792bc0066d912cd91da3e36a

#### reference patch

- bytes: sha256:85185c340edd302f0a3f4e2d2b18d022393f695966e4eb4ea300c263765e81ea

## Surviving mutants: the tests must kill these

- distro.go:22: 8472 -> 8473 | Description=Reserve UDP 8472 for flannel under WSL mirrored networking

Kill each swap where the line is behaviour with a test in a new test file named like the repo's tests (`<pkg>/<name>_test.go` beside the code); you may add new test files but must not edit existing ones. For each swap you do not test, write one commit-message line `mutant-skip: <path>:<line> <comment|doc|message|type|equivalent>`. Use `equivalent` only for a swap that cannot change behaviour or a line no test can reach (an error path of an os/exec pipe); a plain behaviour line a test can reach, such as a failing writer or a bad input, is killed with a test, never skipped as `equivalent`.
