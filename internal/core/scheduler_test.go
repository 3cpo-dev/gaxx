package core

import "testing"

// TestChunkInputs tests the ChunkInputs function
func TestChunkInputs(t *testing.T) {
	in := []string{"a", "b", "c", "d", "e"}
	chunks := ChunkInputs(in, 2)
	if len(chunks) != 3 {
		t.Fatalf("expected 3 chunks, got %d", len(chunks))
	}
	if len(chunks[0]) != 2 || chunks[0][0] != "a" || chunks[0][1] != "b" {
		t.Fatalf("unexpected first chunk")
	}
	if len(chunks[2]) != 1 || chunks[2][0] != "e" {
		t.Fatalf("unexpected last chunk")
	}
}
