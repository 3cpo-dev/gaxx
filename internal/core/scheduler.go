package core

// ChunkInputs splits a list of inputs into chunks of at most chunkSize.
func ChunkInputs(inputs []string, chunkSize int) [][]string {
	if chunkSize <= 0 {
		return [][]string{inputs}
	}
	var chunks [][]string
	for i := 0; i < len(inputs); i += chunkSize {
		end := i + chunkSize
		if end > len(inputs) {
			end = len(inputs)
		}
		chunks = append(chunks, inputs[i:end])
	}
	return chunks
}
