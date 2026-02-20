package api

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
)

// ReadJSONFile reads a JSON file from disk and unmarshals it into a Go data structure.
// Returns the data as interface{} (can be map or slice) or an error.
func ReadJSONFile(filePath string) (interface{}, error) {
	data, err := ioutil.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("cannot read file %s: %w", filePath, err)
	}

	var jsonData interface{}
	if err := json.Unmarshal(data, &jsonData); err != nil {
		return nil, fmt.Errorf("invalid JSON in %s: %w", filePath, err)
	}

	return jsonData, nil
}
