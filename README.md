# Gaxx
Gaxx is a tool that quickly rents and coordinates temporary cloud servers to run lots of small jobs in parallel. It installs a tiny helper on each server, splits your input into chunks, runs them at the same time, then merges the results, with sensible security on by default.

# What is it?
Gaxx is a distributed task runner for short-lived fleets of VPS machines. It’s designed to “burst” up a large pool of servers for a short time, finish the work fast, and shut them down to save cost and reduce manual ops.

# How it works?
Spins up new VPS nodes or connects to ones you already have.

Installs a lightweight agent on each node that listens for work.

Splits your input into manageable chunks and fans them out as tasks.

Runs tasks in parallel across all nodes to finish quickly.

Collects and stitches all outputs back into a single result.

Tears down or detaches from nodes when the job is done.

# Why it’s useful?
Faster batch work by parallelizing across many temporary servers.

Lower cost by paying only for the burst window instead of keeping machines idle.

Less hassle: provisioning, distribution, and result collection are handled for you.

Safer by default: comes with secure settings so you don’t have to hand-roll hardening.

# Practical case:
You need to process 1 million URLs. Gaxx spins up a few hundred VPS instances, deploys its agent, splits the URL list into chunks, runs the fetch/parse tasks simultaneously, and returns one combined output file—then the fleet is torn down.


## Features
- Fleet-based execution with ephemeral keys and hardened SSH
- Agent-based command execution with streaming logs and heartbeats
- YAML task modules with variable substitution and input chunking
- SQLite-backed state: runs, nodes, artifacts
- Structured JSON logging

## Usage
```
Gaxx [command]

Available Commands:
  delete      Delete an existing fleet or even a single box
  help        Help about any command
  images      Show image options
  init        gaxx initialization command. Run this the first time.
  ls          List running boxes
  run         Send a command to a fleet
  scan        Send a command to a fleet, but also with files upload & chunks splitting
  scp         Send a file/folder to a fleet using SCP
  spawn       Spawn a fleet or even a single box
  ssh         Start SSH terminal for a box

Flags:
      --config string   config file
  -h, --help            help for gaxx
  -l, --log string      Set log level. Available: debug, info, warn, error, fatal (default "info")
      --proxy string    HTTP Proxy (Useful for debugging. Example: http://127.0.0.1:8080)
  -t, --toggle          Help message for toggle
```

## Quickstart
```bash
Ensure the binary is on PATH:
export PATH="$(go env GOPATH)/bin:$PATH"
gaxx -h
gaxx init
```
Note: init currently runs but is a stub (“init not yet implemented”).


1. Install (requires Go):
```bash
go install github.com/3cpo-dev/gaxx/cmd/gaxx@latest
go install github.com/3cpo-dev/gaxx/cmd/gaxx-agent@latest
```
2. Initialize once:
```bash
gaxx init
```
3. Create a fleet:
```bash
gaxx spawn --provider linode --count 3 --name alpha --region us-east --image linode/ubuntu22.04 --size g6-nanode-1
```
4. Run a command:
```bash
gaxx run --name alpha -- echo "hello"
```
5. Transfer files:
```bash
gaxx scp --name alpha --push ./wordlist.txt:/home/gx/wordlist.txt
```
6. SSH into a box:
```bash
gaxx ssh --name alpha --node alpha-1
```
7. Delete:
```bash
gaxx delete --name alpha
```

## Security defaults
- SSH key-only, `PermitRootLogin no`, `PasswordAuthentication no`
- Non-root user (`gx` by default)
- Strict `known_hosts` management
- Retries, backoff, and timeouts for all remote ops

## Development
- Go 1.21+
- Build: `make build`
- Test: `make test`
- Snapshot release: `make release-snapshot`

## License
Apache-2.0


