package client

import (
	"encoding/json"
	"io"
	"strings"
)

// parseJSONObjects extracts complete JSON objects from a string and returns:
// 1. A slice of complete JSON objects
// 2. Any remaining unparsed data
// 3. An error if there was a problem parsing
func parseJSONObjects(input string) ([]string, string, error) {
	jsonObjects := []string{}

	// Handle empty or whitespace-only input
	trimmedInput := strings.TrimSpace(input)
	if trimmedInput == "" {
		return jsonObjects, "", nil
	}

	// Create a reader from the input string
	reader := strings.NewReader(input)
	decoder := json.NewDecoder(reader)

	// Track how much we've processed
	var processedBytes int64 = 0

	// Loop to extract all complete JSON objects
	for {
		// Try to decode one complete JSON value
		var raw json.RawMessage
		err := decoder.Decode(&raw)

		if err == io.EOF {
			// We've reached the end of the input
			break
		}

		if err != nil {
			// Return what we've parsed so far, the remaining unparsed data, and the error
			remaining := strings.TrimSpace(input[processedBytes:])
			return jsonObjects, remaining, err
		}

		// We successfully decoded a complete JSON object
		jsonObjects = append(jsonObjects, string(raw))

		// Update how many bytes we've processed
		processedBytes = decoder.InputOffset()
	}

	// Check if there's any remaining data after whitespace
	remaining := ""
	if processedBytes < int64(len(input)) {
		// Get the remaining data and trim whitespace
		remaining = strings.TrimSpace(input[processedBytes:])

		// If there's non-whitespace content remaining, it's garbage
		if remaining != "" {
			return jsonObjects, remaining, io.ErrUnexpectedEOF
		}
	}

	return jsonObjects, remaining, nil
}
