package tests

import (
	"os"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestSwaggerYAMLIsValid(t *testing.T) {
	data, err := os.ReadFile("../../docs/swagger.yaml")
	if err != nil {
		t.Fatalf("read swagger yaml: %v", err)
	}

	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		t.Fatalf("parse swagger yaml: %v", err)
	}
	if document["openapi"] == nil || document["paths"] == nil {
		t.Fatal("swagger yaml must define openapi and paths")
	}
}
