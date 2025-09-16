package yaml_test

import (
	"fmt"
	"strings"
	"testing"

	"gopkg.in/yaml.v3"
)

func TestCheckSequence(t *testing.T) {
	yaml.PreserveBlankLines = true
	defer func() {
		yaml.PreserveBlankLines = false
	}()

	input := `items:
  - item1

  - item2

  - item3`

	var node yaml.Node
	err := yaml.Unmarshal([]byte(input), &node)
	if err != nil {
		t.Fatalf("Failed to unmarshal: %v", err)
	}

	// Print the node structure
	fmt.Println("=== Node Structure ===")
	printNodeDebug(&node, 0)

	// Marshal back
	output, err := yaml.Marshal(&node)
	if err != nil {
		t.Fatalf("Failed to marshal: %v", err)
	}

	fmt.Printf("\n=== Output ===\n%s", string(output))

	// Count blank lines in output
	lines := strings.Split(string(output), "\n")
	blankCount := 0
	for _, line := range lines {
		if line == "" {
			blankCount++
		}
	}
	fmt.Printf("\n=== Found %d blank lines in output ===\n", blankCount)

	// Check if we have blank lines between items
	outputStr := string(output)
	if strings.Contains(outputStr, "\n\n") {
		fmt.Println("Output contains consecutive newlines (blank lines)")
	}
}

func printNodeDebug(n *yaml.Node, depth int) {
	indent := ""
	for i := 0; i < depth; i++ {
		indent += "  "
	}
	fmt.Printf("%sKind=%v, Value=%q, BlankLinesBefore=%d\n",
		indent, n.Kind, n.Value, n.BlankLinesBefore)
	for _, child := range n.Content {
		printNodeDebug(child, depth+1)
	}
}