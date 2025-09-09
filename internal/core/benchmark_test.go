package core

import (
	"context"
	"testing"
	"time"
)

func BenchmarkMetricsRecording(b *testing.B) {
	metrics := NewMetrics()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		metrics.RecordRequest(time.Millisecond)
		if i%10 == 0 {
			metrics.RecordError()
		}
	}
}

func BenchmarkBuildCommand(b *testing.B) {
	config := &Config{
		Provider:    "test",
		SSHKeyPath:  "/tmp/test-key",
		Concurrency: 5,
	}

	mockProvider := &MockProvider{}
	gaxx := NewGaxx(config, mockProvider)

	task := Task{
		Command: "echo",
		Args:    []string{"hello", "world", "test", "benchmark"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = gaxx.BuildCommand(task)
	}
}

// Memory allocation benchmarks
func BenchmarkMemoryAllocation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		config := &Config{
			Provider:    "test",
			SSHKeyPath:  "/tmp/test-key",
			Concurrency: 10,
		}

		mockProvider := &MockProvider{}
		gaxx := NewGaxx(config, mockProvider)

		// Simulate some operations
		_, _, _ = gaxx.GetMetrics()
	}
}

func BenchmarkProviderOperations(b *testing.B) {
	mockProvider := &MockProvider{}
	ctx := context.Background()

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// Test provider operations
		instances, err := mockProvider.CreateInstances(ctx, 3, "benchmark")
		if err != nil {
			b.Fatalf("CreateInstances failed: %v", err)
		}

		_, err = mockProvider.ListInstances(ctx, "benchmark")
		if err != nil {
			b.Fatalf("ListInstances failed: %v", err)
		}

		err = mockProvider.DeleteInstances(ctx, "benchmark")
		if err != nil {
			b.Fatalf("DeleteInstances failed: %v", err)
		}

		_ = instances // Use the result
	}
}

func BenchmarkConcurrentMetrics(b *testing.B) {
	metrics := NewMetrics()

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			metrics.RecordRequest(time.Millisecond)
			if pb.Next() {
				metrics.RecordError()
			}
		}
	})
}

func BenchmarkConfigCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		config := &Config{
			Provider:    "test",
			Token:       "test-token",
			Region:      "us-east",
			SSHKeyPath:  "/tmp/test-key",
			Monitoring:  true,
			Concurrency: 10,
		}

		_ = config // Use the result
	}
}

func BenchmarkInstanceCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		instance := Instance{
			ID:   "test-1",
			Name: "test-instance",
			IP:   "192.168.1.100",
			User: "gx",
			Port: 22,
		}

		_ = instance // Use the result
	}
}

func BenchmarkTaskCreation(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		task := Task{
			Command: "echo",
			Args:    []string{"hello", "world"},
			Env:     map[string]string{"TEST": "value"},
			Input:   "test input",
		}

		_ = task // Use the result
	}
}
