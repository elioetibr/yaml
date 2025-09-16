package yaml

import (
	"bytes"
	"fmt"
	"testing"
)

func TestSequenceFlow(t *testing.T) {
	PreserveBlankLines = true
	defer func() {
		PreserveBlankLines = false
	}()

	// Simple sequence with blank line
	input := []byte(`- item1

- item2`)

	// Parse
	p := parser{}
	yaml_parser_initialize(&p.parser)
	yaml_parser_set_input_string(&p.parser, input)
	p.preserveBlankLines = true
	p.parser.preserve_blank_lines = true

	// Collect all events
	var events []yaml_event_t
	for {
		var event yaml_event_t
		if !yaml_parser_parse(&p.parser, &event) {
			break
		}
		events = append(events, event)
		if event.typ == yaml_STREAM_END_EVENT {
			break
		}
	}

	// Print events
	fmt.Println("=== Events ===")
	for i, event := range events {
		if event.typ == yaml_SCALAR_EVENT || event.typ == yaml_SEQUENCE_START_EVENT {
			fmt.Printf("Event %d: type=%v, value=%q, blank_lines_before=%d\n",
				i, event.typ, string(event.value), event.blank_lines_before)
		}
	}

	// Now emit the events
	var buf bytes.Buffer
	emitter := yaml_emitter_t{}
	yaml_emitter_initialize(&emitter)
	yaml_emitter_set_output_writer(&emitter, &buf)
	emitter.preserve_blank_lines = true

	fmt.Println("\n=== Emitting ===")
	for i, event := range events {
		eventCopy := event // Make a copy

		// Debug: print before emitting
		if event.typ == yaml_SCALAR_EVENT {
			fmt.Printf("Emitting scalar %q with blank_lines_before=%d\n",
				string(event.value), event.blank_lines_before)
		}

		if !yaml_emitter_emit(&emitter, &eventCopy) {
			t.Fatalf("Failed to emit event %d", i)
		}
	}

	output := buf.String()
	fmt.Printf("\n=== Output ===\n%s", output)

	// Check for blank line (with document marker)
	expected := "---\n- item1\n\n- item2\n"
	if output != expected {
		// Also accept without document marker for compatibility
		expectedAlt := "- item1\n\n- item2\n"
		if output != expectedAlt {
			t.Errorf("Expected:\n%s\nGot:\n%s", expected, output)
		}
	}
}