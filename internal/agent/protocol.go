package agent

import "time"

// HeartbeatRequest is empty; the agent reports itself.
type HeartbeatRequest struct{}

type HeartbeatResponse struct {
	Time    time.Time `json:"time"`
	Host    string    `json:"host"`
	Version string    `json:"version"`
}

type ExecRequest struct {
	Command string   `json:"command"`
	Args    []string `json:"args"`
	Env     []string `json:"env"`
	Timeout int      `json:"timeout_seconds"`
	WorkDir string   `json:"work_dir"`
	Input   string   `json:"input"`
}

type ExecResponse struct {
	ExitCode int    `json:"exit_code"`
	Stdout   string `json:"stdout"`
	Stderr   string `json:"stderr"`
	Duration int64  `json:"duration_ms"`
}
