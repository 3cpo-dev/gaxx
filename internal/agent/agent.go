package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"os/exec"
	"time"

	"github.com/3cpo-dev/gaxx/internal/telemetry"
)

type Server struct {
	Version string
	srv     *http.Server
}

// Routes for the server
func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		_ = r.Body.Close()

		// Record heartbeat metrics
		telemetry.CounterGlobal("gaxx_agent_heartbeats", 1, map[string]string{
			"component": "agent",
			"endpoint":  "heartbeat",
		})

		h := HeartbeatResponse{Time: time.Now(), Host: r.Host, Version: s.Version}
		_ = json.NewEncoder(w).Encode(h)

		telemetry.TimerGlobal("gaxx_agent_request_duration", time.Since(start), map[string]string{
			"component": "agent",
			"endpoint":  "heartbeat",
			"status":    "200",
		})
	})
	mux.HandleFunc("/v0/exec", func(w http.ResponseWriter, r *http.Request) {
		// Optional token-based auth via env var
		if tok := os.Getenv("GAXX_AGENT_TOKEN"); tok != "" {
			auth := r.Header.Get("Authorization")
			x := r.Header.Get("X-Auth-Token")
			if auth != "Bearer "+tok && x != tok {
				http.Error(w, "unauthorized", http.StatusUnauthorized)
				return
			}
		}

		requestStart := time.Now()
		defer r.Body.Close()

		var req ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			telemetry.CounterGlobal("gaxx_agent_exec_errors", 1, map[string]string{
				"component": "agent",
				"endpoint":  "exec",
				"error":     "decode_request",
			})
			http.Error(w, err.Error(), 400)
			return
		}

		// Record exec request
		telemetry.CounterGlobal("gaxx_agent_exec_requests", 1, map[string]string{
			"component": "agent",
			"endpoint":  "exec",
			"command":   req.Command,
		})

		ctx := r.Context()
		if req.Timeout > 0 {
			var cancel context.CancelFunc
			ctx, cancel = context.WithTimeout(ctx, time.Duration(req.Timeout)*time.Second)
			defer cancel()
		}

		cmd := exec.CommandContext(ctx, req.Command, req.Args...)
		if req.WorkDir != "" {
			cmd.Dir = req.WorkDir
		}
		if len(req.Env) > 0 {
			cmd.Env = append(cmd.Env, req.Env...)
		}

		execStart := time.Now()
		out, err := cmd.CombinedOutput()
		execDuration := time.Since(execStart)

		resp := ExecResponse{Stdout: string(out), Stderr: "", Duration: execDuration.Milliseconds()}
		status := "success"

		if err != nil {
			status = "error"
			if exit, ok := err.(*exec.ExitError); ok {
				resp.ExitCode = exit.ExitCode()
			} else {
				resp.ExitCode = 1
			}
		}

		// Record execution metrics
		labels := map[string]string{
			"component": "agent",
			"endpoint":  "exec",
			"command":   req.Command,
			"status":    status,
		}

		telemetry.TimerGlobal("gaxx_agent_exec_duration", execDuration, labels)
		telemetry.TimerGlobal("gaxx_agent_request_duration", time.Since(requestStart), labels)
		telemetry.HistogramGlobal("gaxx_agent_exec_output_size", float64(len(out)), labels)

		if status == "success" {
			telemetry.CounterGlobal("gaxx_agent_exec_successful", 1, labels)
		} else {
			telemetry.CounterGlobal("gaxx_agent_exec_failed", 1, labels)
		}

		_ = json.NewEncoder(w).Encode(resp)
	})
}

// ListenAndServe starts the server
func (s *Server) ListenAndServe(addr string) error {
	mux := http.NewServeMux()
	s.routes(mux)
	s.srv = &http.Server{Addr: addr, Handler: mux}
	return s.srv.ListenAndServe()
}

// Shutdown the server
func (s *Server) Shutdown(ctx context.Context) error {
	if s.srv == nil {
		return fmt.Errorf("server not running")
	}
	return s.srv.Shutdown(ctx)
}
