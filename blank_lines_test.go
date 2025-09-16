package yaml_test

import (
	"bytes"
	"strings"
	"testing"

	"github.com/elioetibr/yaml"
)

func TestBlankLinePreservation(t *testing.T) {
	// Save original flag state
	originalFlag := yaml.PreserveBlankLines
	defer func() {
		yaml.PreserveBlankLines = originalFlag
	}()

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name: "simple mapping with blank lines",
			input: `key1: value1

key2: value2


key3: value3
`,
			expected: `key1: value1

key2: value2


key3: value3
`,
		},
		{
			name: "sequence with blank lines",
			input: `items:
  - item1

  - item2

  - item3
`,
			expected: `items:
  - item1

  - item2

  - item3
`,
		},
		{
			name: "mixed with comments and blank lines",
			input: `# Header comment
key1: value1

# Comment for key2
key2: value2

# Comment for key3

key3: value3
`,
			expected: `# Header comment
key1: value1

# Comment for key2
key2: value2

# Comment for key3

key3: value3
`,
		},
		{
			name: "nested structures with blank lines",
			input: `parent:
  child1: value1

  child2: value2

  nested:
    - item1

    - item2
`,
			expected: `parent:
  child1: value1

  child2: value2

  nested:
    - item1

    - item2
`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test with feature flag enabled
			yaml.PreserveBlankLines = true

			// Parse the input
			var node yaml.Node
			err := yaml.Unmarshal([]byte(tt.input), &node)
			if err != nil {
				t.Fatalf("Failed to unmarshal: %v", err)
			}

			// Encode back to YAML
			var buf bytes.Buffer
			encoder := yaml.NewEncoder(&buf)
			encoder.SetIndent(2) // Use 2-space indentation to match input
			encoder.SetPreserveBlankLines(true)
			err = encoder.Encode(&node)
			if err != nil {
				t.Fatalf("Failed to encode: %v", err)
			}
			encoder.Close()

			// Compare output
			output := buf.String()
			if output != tt.expected {
				t.Errorf("Blank lines not preserved.\nExpected:\n%s\nGot:\n%s", tt.expected, output)
			}
		})
	}
}

func TestBlankLinePreservationDisabled(t *testing.T) {
	// Save original flag state
	originalFlag := yaml.PreserveBlankLines
	defer func() {
		yaml.PreserveBlankLines = originalFlag
	}()

	input := `key1: value1

key2: value2


key3: value3
`

	// Test with feature flag disabled
	yaml.PreserveBlankLines = false

	var node yaml.Node
	err := yaml.Unmarshal([]byte(input), &node)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetPreserveBlankLines(false)
	err = encoder.Encode(&node)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}
	encoder.Close()

	// With flag disabled, blank lines should not be preserved
	output := buf.String()
	if strings.Count(output, "\n\n") > 0 {
		t.Errorf("Blank lines were preserved when feature was disabled.\nGot:\n%s", output)
	}
}

func TestPerInstanceControl(t *testing.T) {
	// Save original flag state
	originalFlag := yaml.PreserveBlankLines
	defer func() {
		yaml.PreserveBlankLines = originalFlag
	}()

	input := `key1: value1

key2: value2
`

	// Test that per-instance setting overrides global
	yaml.PreserveBlankLines = false

	// Decoder with preservation enabled
	decoder := yaml.NewDecoder(strings.NewReader(input))
	decoder.SetPreserveBlankLines(true)

	var node yaml.Node
	err := decoder.Decode(&node)
	if err != nil {
		t.Fatalf("Failed to decode: %v", err)
	}

	// Verify blank lines were tracked
	if node.Content[0].Content[2].BlankLinesBefore != 1 {
		t.Errorf("Expected BlankLinesBefore=1, got %d", node.Content[0].Content[2].BlankLinesBefore)
	}

	// Encoder with preservation enabled
	var buf bytes.Buffer
	encoder := yaml.NewEncoder(&buf)
	encoder.SetPreserveBlankLines(true)
	err = encoder.Encode(&node)
	if err != nil {
		t.Fatalf("Failed to encode: %v", err)
	}
	encoder.Close()

	// Should have preserved the blank line
	output := buf.String()
	if !strings.Contains(output, "\n\nkey2:") {
		t.Errorf("Blank line not preserved with per-instance setting.\nGot:\n%s", output)
	}
}

func BenchmarkBlankLinePreservation(b *testing.B) {
	input := `key1: value1

key2: value2

key3:
  nested1: value3

  nested2: value4
`

	b.Run("WithPreservation", func(b *testing.B) {
		yaml.PreserveBlankLines = true
		for i := 0; i < b.N; i++ {
			var node yaml.Node
			yaml.Unmarshal([]byte(input), &node)
			yaml.Marshal(&node)
		}
	})

	b.Run("WithoutPreservation", func(b *testing.B) {
		yaml.PreserveBlankLines = false
		for i := 0; i < b.N; i++ {
			var node yaml.Node
			yaml.Unmarshal([]byte(input), &node)
			yaml.Marshal(&node)
		}
	})
}
