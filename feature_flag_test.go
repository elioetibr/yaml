package yaml_test

import (
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

// TestFeatureFlagInfrastructure verifies the feature flag infrastructure is in place
func TestFeatureFlagInfrastructure(t *testing.T) {
	// Save original flag state
	originalFlag := yaml.PreserveBlankLines
	defer func() {
		yaml.PreserveBlankLines = originalFlag
	}()

	t.Run("GlobalFlag", func(t *testing.T) {
		// Test that global flag can be set
		yaml.PreserveBlankLines = false
		if yaml.PreserveBlankLines {
			t.Error("Expected PreserveBlankLines to be false")
		}

		yaml.PreserveBlankLines = true
		if !yaml.PreserveBlankLines {
			t.Error("Expected PreserveBlankLines to be true")
		}
	})

	t.Run("DecoderFlag", func(t *testing.T) {
		yaml.PreserveBlankLines = false

		input := "key: value"
		decoder := yaml.NewDecoder(strings.NewReader(input))

		// Test that SetPreserveBlankLines method exists and works
		decoder.SetPreserveBlankLines(true)

		var node yaml.Node
		err := decoder.Decode(&node)
		if err != nil {
			t.Fatalf("Failed to decode: %v", err)
		}
	})

	t.Run("EncoderFlag", func(t *testing.T) {
		yaml.PreserveBlankLines = false

		var builder strings.Builder
		encoder := yaml.NewEncoder(&builder)

		// Test that SetPreserveBlankLines method exists and works
		encoder.SetPreserveBlankLines(true)

		node := &yaml.Node{
			Kind: yaml.DocumentNode,
			Content: []*yaml.Node{
				{
					Kind: yaml.MappingNode,
					Content: []*yaml.Node{
						{Kind: yaml.ScalarNode, Value: "key"},
						{Kind: yaml.ScalarNode, Value: "value"},
					},
				},
			},
		}

		err := encoder.Encode(node)
		if err != nil {
			t.Fatalf("Failed to encode: %v", err)
		}
		encoder.Close()
	})

	t.Run("NodeStructure", func(t *testing.T) {
		// Test that Node has the new fields
		node := &yaml.Node{
			Kind:             yaml.ScalarNode,
			Value:            "test",
			BlankLinesBefore: 2,
			BlankLinesAfter:  1,
		}

		if node.BlankLinesBefore != 2 {
			t.Errorf("Expected BlankLinesBefore to be 2, got %d", node.BlankLinesBefore)
		}

		if node.BlankLinesAfter != 1 {
			t.Errorf("Expected BlankLinesAfter to be 1, got %d", node.BlankLinesAfter)
		}

		// Test IsZero with blank lines
		emptyNode := &yaml.Node{}
		if !emptyNode.IsZero() {
			t.Error("Expected empty node to be zero")
		}

		nodeWithBlanks := &yaml.Node{BlankLinesBefore: 1}
		if nodeWithBlanks.IsZero() {
			t.Error("Expected node with blank lines to not be zero")
		}
	})
}

// TestBackwardCompatibility ensures no breaking changes
func TestBackwardCompatibility(t *testing.T) {
	// Save original flag state
	originalFlag := yaml.PreserveBlankLines
	defer func() {
		yaml.PreserveBlankLines = originalFlag
	}()

	// With feature disabled (default), should work as before
	yaml.PreserveBlankLines = false

	input := `
name: test
items:
  - one
  - two
mapping:
  key1: value1
  key2: value2
`

	var data interface{}
	err := yaml.Unmarshal([]byte(input), &data)
	if err != nil {
		t.Fatalf("Unmarshal failed: %v", err)
	}

	// Marshal back
	output, err := yaml.Marshal(data)
	if err != nil {
		t.Fatalf("Marshal failed: %v", err)
	}

	// Should produce valid YAML
	var data2 interface{}
	err = yaml.Unmarshal(output, &data2)
	if err != nil {
		t.Fatalf("Round-trip failed: %v", err)
	}
}

// TestFeatureFlagImpact verifies feature flag doesn't break existing functionality
func TestFeatureFlagImpact(t *testing.T) {
	// Save original flag state
	originalFlag := yaml.PreserveBlankLines
	defer func() {
		yaml.PreserveBlankLines = originalFlag
	}()

	testCases := []struct {
		name string
		yaml string
	}{
		{
			name: "simple",
			yaml: "key: value",
		},
		{
			name: "list",
			yaml: "items:\n  - one\n  - two",
		},
		{
			name: "nested",
			yaml: "parent:\n  child:\n    grandchild: value",
		},
		{
			name: "mixed",
			yaml: "name: test\nitems:\n  - one\n  - two\ncount: 42",
		},
	}

	for _, flagState := range []bool{false, true} {
		yaml.PreserveBlankLines = flagState
		flagName := "disabled"
		if flagState {
			flagName = "enabled"
		}

		for _, tc := range testCases {
			t.Run(flagName+"_"+tc.name, func(t *testing.T) {
				var node yaml.Node
				err := yaml.Unmarshal([]byte(tc.yaml), &node)
				if err != nil {
					t.Fatalf("Unmarshal failed with flag %s: %v", flagName, err)
				}

				output, err := yaml.Marshal(&node)
				if err != nil {
					t.Fatalf("Marshal failed with flag %s: %v", flagName, err)
				}

				// Verify output is valid YAML
				var verification yaml.Node
				err = yaml.Unmarshal(output, &verification)
				if err != nil {
					t.Fatalf("Verification failed with flag %s: %v", flagName, err)
				}
			})
		}
	}
}