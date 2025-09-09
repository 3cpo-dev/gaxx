package core

import (
	"context"
	"fmt"
	"strings"
	"testing"
	"time"
)

// MockProvider for testing
type MockProvider struct {
	instances []Instance
}

func (m *MockProvider) CreateInstances(ctx context.Context, count int, name string) ([]Instance, error) {
	instances := make([]Instance, count)
	for i := 0; i < count; i++ {
		instances[i] = Instance{
			ID:   fmt.Sprintf("mock-%d", i+1),
			Name: fmt.Sprintf("%s-%d", name, i+1),
			IP:   fmt.Sprintf("192.168.1.%d", 100+i),
			User: "gx",
			Port: 22,
		}
	}
	m.instances = append(m.instances, instances...)
	return instances, nil
}

func (m *MockProvider) DeleteInstances(ctx context.Context, name string) error {
	var remaining []Instance
	for _, inst := range m.instances {
		if name == "" || !strings.HasPrefix(inst.Name, name) {
			remaining = append(remaining, inst)
		}
	}
	m.instances = remaining
	return nil
}

func (m *MockProvider) ListInstances(ctx context.Context, name string) ([]Instance, error) {
	var result []Instance
	for _, inst := range m.instances {
		if name == "" || strings.HasPrefix(inst.Name, name) {
			result = append(result, inst)
		}
	}
	return result, nil
}

func TestProviderInterface(t *testing.T) {
	mockProvider := &MockProvider{}

	ctx := context.Background()

	// Test CreateInstances
	instances, err := mockProvider.CreateInstances(ctx, 2, "test")
	if err != nil {
		t.Fatalf("CreateInstances failed: %v", err)
	}

	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}

	// Test ListInstances
	listResult, err := mockProvider.ListInstances(ctx, "test")
	if err != nil {
		t.Fatalf("ListInstances failed: %v", err)
	}

	if len(listResult) != 2 {
		t.Errorf("Expected 2 instances in list, got %d", len(listResult))
	}

	// Test DeleteInstances
	err = mockProvider.DeleteInstances(ctx, "test")
	if err != nil {
		t.Fatalf("DeleteInstances failed: %v", err)
	}

	// Verify deletion
	listResult, err = mockProvider.ListInstances(ctx, "test")
	if err != nil {
		t.Fatalf("ListInstances after delete failed: %v", err)
	}

	if len(listResult) != 0 {
		t.Errorf("Expected 0 instances after deletion, got %d", len(listResult))
	}
}

func TestConfig(t *testing.T) {
	config := &Config{
		Provider:    "test",
		Token:       "test-token",
		Region:      "us-east",
		SSHKeyPath:  "/tmp/test-key",
		Monitoring:  true,
		Concurrency: 10,
	}

	if config.Provider != "test" {
		t.Errorf("Expected provider 'test', got '%s'", config.Provider)
	}

	if config.Concurrency != 10 {
		t.Errorf("Expected concurrency 10, got %d", config.Concurrency)
	}
}

func TestInstance(t *testing.T) {
	instance := Instance{
		ID:   "test-1",
		Name: "test-instance",
		IP:   "192.168.1.100",
		User: "gx",
		Port: 22,
	}

	if instance.ID != "test-1" {
		t.Errorf("Expected ID 'test-1', got '%s'", instance.ID)
	}

	if instance.IP != "192.168.1.100" {
		t.Errorf("Expected IP '192.168.1.100', got '%s'", instance.IP)
	}
}

func TestTask(t *testing.T) {
	task := Task{
		Command: "echo",
		Args:    []string{"hello", "world"},
		Env:     map[string]string{"TEST": "value"},
		Input:   "test input",
	}

	if task.Command != "echo" {
		t.Errorf("Expected command 'echo', got '%s'", task.Command)
	}

	if len(task.Args) != 2 {
		t.Errorf("Expected 2 args, got %d", len(task.Args))
	}

	if task.Env["TEST"] != "value" {
		t.Errorf("Expected env TEST='value', got '%s'", task.Env["TEST"])
	}
}

func TestMetrics(t *testing.T) {
	metrics := NewMetrics()

	// Record some metrics
	metrics.RecordRequest(100 * time.Millisecond)
	metrics.RecordRequest(200 * time.Millisecond)
	metrics.RecordError()

	requests, errors, duration := metrics.GetStats()

	if requests != 2 {
		t.Errorf("Expected 2 requests, got %d", requests)
	}

	if errors != 1 {
		t.Errorf("Expected 1 error, got %d", errors)
	}

	if duration != 300*time.Millisecond {
		t.Errorf("Expected 300ms duration, got %v", duration)
	}
}

func TestBuildCommand(t *testing.T) {
	config := &Config{
		Provider:    "test",
		SSHKeyPath:  "/tmp/test-key",
		Concurrency: 5,
	}

	mockProvider := &MockProvider{}
	gaxx := NewGaxx(config, mockProvider)

	task := Task{
		Command: "echo",
		Args:    []string{"hello", "world"},
	}

	cmd := gaxx.BuildCommand(task)
	expected := "echo hello world"

	if cmd != expected {
		t.Errorf("Expected command '%s', got '%s'", expected, cmd)
	}
}
