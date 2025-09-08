package agent

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os/exec"
	"time"
)

type Server struct {
	Version string
	srv     *http.Server
}

// Routes for the server
func (s *Server) routes(mux *http.ServeMux) {
	mux.HandleFunc("/v0/heartbeat", func(w http.ResponseWriter, r *http.Request) {
		_ = r.Body.Close()
		h := HeartbeatResponse{Time: time.Now(), Host: r.Host, Version: s.Version}
		_ = json.NewEncoder(w).Encode(h)
	})
	mux.HandleFunc("/v0/exec", func(w http.ResponseWriter, r *http.Request) {
		defer r.Body.Close()
		var req ExecRequest
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, err.Error(), 400)
			return
		}
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
		start := time.Now()
		out, err := cmd.CombinedOutput()
		dur := time.Since(start)
		resp := ExecResponse{Stdout: string(out), Stderr: "", Duration: dur.Milliseconds()}
		if err != nil {
			if exit, ok := err.(*exec.ExitError); ok {
				resp.ExitCode = exit.ExitCode()
			} else {
				resp.ExitCode = 1
			}
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
