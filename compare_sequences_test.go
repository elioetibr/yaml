package yaml_test

import (
	"fmt"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCompareSequences(t *testing.T) {
	yaml.PreserveBlankLines = true
	defer func() {
		yaml.PreserveBlankLines = false
	}()

	// Test 1: Simple sequence
	input1 := `- item1

- item2`

	var node1 yaml.Node
	err := yaml.Unmarshal([]byte(input1), &node1)
	if err != nil {
		t.Fatalf("Failed to unmarshal simple: %v", err)
	}

	output1, err := yaml.Marshal(&node1)
	if err != nil {
		t.Fatalf("Failed to marshal simple: %v", err)
	}

	fmt.Println("=== Simple Sequence ===")
	fmt.Printf("Input:\n%s\n", input1)
	fmt.Printf("Output:\n%s\n", string(output1))

	// Test 2: Nested sequence
	input2 := `items:
  - item1

  - item2`

	var node2 yaml.Node
	err = yaml.Unmarshal([]byte(input2), &node2)
	if err != nil {
		t.Fatalf("Failed to unmarshal nested: %v", err)
	}

	output2, err := yaml.Marshal(&node2)
	if err != nil {
		t.Fatalf("Failed to marshal nested: %v", err)
	}

	fmt.Println("=== Nested Sequence ===")
	fmt.Printf("Input:\n%s\n", input2)
	fmt.Printf("Output:\n%s\n", string(output2))

	// Print node structure for nested
	fmt.Println("=== Nested Node Structure ===")
	printNodeStructure(&node2, 0)
}

func printNodeStructure(node *yaml.Node, indent int) {
	prefix := ""
	for i := 0; i < indent; i++ {
		prefix += "  "
	}
	fmt.Printf("%sKind=%d, Value=%q, BlankLinesBefore=%d\n",
		prefix, node.Kind, node.Value, node.BlankLinesBefore)
	for _, child := range node.Content {
		printNodeStructure(child, indent+1)
	}
}