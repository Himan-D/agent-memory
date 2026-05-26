package evaluation

import (
	"fmt"
	"os"
	"path/filepath"
)

// EnsureESMemEvalDataset checks that the ES-MemEval dataset file exists at the expected path.
// Returns the path to the dataset.json if found.
func EnsureESMemEvalDataset() (string, error) {
	p := filepath.Join("evaluation", "es_memeval", "dataset.json")
	if _, err := os.Stat(p); os.IsNotExist(err) {
		return "", fmt.Errorf("es_memeval dataset not found at %s", p)
	} else if err != nil {
		return "", fmt.Errorf("failed to stat es_memeval dataset: %w", err)
	}
	return p, nil
}

// LoadESMemEvalToBenchmark loads the dataset.json from evaluation/es_memeval and unmarshals
// it into the shared BenchmarkDataset type. This allows reuse of existing benchmark runner.
func LoadESMemEvalToBenchmark() (*BenchmarkDataset, error) {
	p, err := EnsureESMemEvalDataset()
	if err != nil {
		return nil, err
	}
	data, err := os.ReadFile(p)
	if err != nil {
		return nil, fmt.Errorf("read dataset: %w", err)
	}
	var ds BenchmarkDataset
	if err := json.Unmarshal(data, &ds); err != nil {
		return nil, fmt.Errorf("parse dataset: %w", err)
	}
	return &ds, nil
}
