package agent

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

// TestHeartbeat tests the heartbeat endpoint
func TestHeartbeat(t *testing.T) {
	srv := &Server{Version: "test"}
	mux := http.NewServeMux()
	srv.routes(mux)
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodGet, "/v0/heartbeat", nil)
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	var resp HeartbeatResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.Version != "test" {
		t.Fatalf("version mismatch")
	}
}

// TestExec tests the exec endpoint
func TestExec(t *testing.T) {
	srv := &Server{Version: "test"}
	mux := http.NewServeMux()
	srv.routes(mux)
	body, _ := json.Marshal(ExecRequest{Command: "echo", Args: []string{"hello"}})
	rr := httptest.NewRecorder()
	req := httptest.NewRequest(http.MethodPost, "/v0/exec", bytes.NewReader(body))
	mux.ServeHTTP(rr, req)
	if rr.Code != 200 {
		t.Fatalf("status %d", rr.Code)
	}
	var resp ExecResponse
	if err := json.Unmarshal(rr.Body.Bytes(), &resp); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if resp.ExitCode != 0 {
		t.Fatalf("exit code %d", resp.ExitCode)
	}
	if resp.Stdout == "" {
		t.Fatalf("expected stdout")
	}
}
