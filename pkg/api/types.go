package api

// v0 contains public types for early SDK usage.

type TaskSpec struct {
	Name        string            `json:"name" yaml:"name"`
	Description string            `json:"description" yaml:"description"`
	Command     string            `json:"command" yaml:"command"`
	Args        []string          `json:"args" yaml:"args"`
	Env         map[string]string `json:"env" yaml:"env"`
	// Inputs can be file paths or inline lists to be chunked across nodes.
	Inputs    []string `json:"inputs" yaml:"inputs"`
	ChunkSize int      `json:"chunk_size" yaml:"chunk_size"`
}

type FleetSpec struct {
	Name     string            `json:"name" yaml:"name"`
	Provider string            `json:"provider" yaml:"provider"`
	Count    int               `json:"count" yaml:"count"`
	Labels   map[string]string `json:"labels" yaml:"labels"`
}

type RunStatus string

const (
	RunPending   RunStatus = "pending"
	RunRunning   RunStatus = "running"
	RunSucceeded RunStatus = "succeeded"
	RunFailed    RunStatus = "failed"
)
