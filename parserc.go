//
// Copyright (c) 2011-2019 Canonical Ltd
// Copyright (c) 2006-2010 Kirill Simonov
//
// Permission is hereby granted, free of charge, to any person obtaining a copy of
// this software and associated documentation files (the "Software"), to deal in
// the Software without restriction, including without limitation the rights to
// use, copy, modify, merge, publish, distribute, sublicense, and/or sell copies
// of the Software, and to permit persons to whom the Software is furnished to do
// so, subject to the following conditions:
//
// The above copyright notice and this permission notice shall be included in all
// copies or substantial portions of the Software.
//
// THE SOFTWARE IS PROVIDED "AS IS", WITHOUT WARRANTY OF ANY KIND, EXPRESS OR
// IMPLIED, INCLUDING BUT NOT LIMITED TO THE WARRANTIES OF MERCHANTABILITY,
// FITNESS FOR A PARTICULAR PURPOSE AND NONINFRINGEMENT. IN NO EVENT SHALL THE
// AUTHORS OR COPYRIGHT HOLDERS BE LIABLE FOR ANY CLAIM, DAMAGES OR OTHER
// LIABILITY, WHETHER IN AN ACTION OF CONTRACT, TORT OR OTHERWISE, ARISING FROM,
// OUT OF OR IN CONNECTION WITH THE SOFTWARE OR THE USE OR OTHER DEALINGS IN THE
// SOFTWARE.

package yaml

import (
	"bytes"
	// "fmt"
)

// The parser implements the following grammar:
//
// stream               ::= STREAM-START implicit_document? explicit_document* STREAM-END
// implicit_document    ::= block_node DOCUMENT-END*
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
// block_node_or_indentless_sequence    ::=
//                          ALIAS
//                          | properties (block_content | indentless_block_sequence)?
//                          | block_content
//                          | indentless_block_sequence
// block_node           ::= ALIAS
//                          | properties block_content?
//                          | block_content
// flow_node            ::= ALIAS
//                          | properties flow_content?
//                          | flow_content
// properties           ::= TAG ANCHOR? | ANCHOR TAG?
// block_content        ::= block_collection | flow_collection | SCALAR
// flow_content         ::= flow_collection | SCALAR
// block_collection     ::= block_sequence | block_mapping
// flow_collection      ::= flow_sequence | flow_mapping
// block_sequence       ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
// indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
// block_mapping        ::= BLOCK-MAPPING_START
//                          ((KEY block_node_or_indentless_sequence?)?
//                          (VALUE block_node_or_indentless_sequence?)?)*
//                          BLOCK-END
// flow_sequence        ::= FLOW-SEQUENCE-START
//                          (flow_sequence_entry FLOW-ENTRY)*
//                          flow_sequence_entry?
//                          FLOW-SEQUENCE-END
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
// flow_mapping         ::= FLOW-MAPPING-START
//                          (flow_mapping_entry FLOW-ENTRY)*
//                          flow_mapping_entry?
//                          FLOW-MAPPING-END
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?

// Peek the next token in the token queue.
func peek_token(parser *yaml_parser_t) *yaml_token_t {
	if parser.token_available || yaml_parser_fetch_more_tokens(parser) {
		token := &parser.tokens[parser.tokens_head]
		yaml_parser_unfold_comments(parser, token)
		return token
	}
	return nil
}

// yaml_parser_unfold_comments walks through the comments queue and joins all
// comments behind the position of the provided token into the respective
// top-level comment slices in the parser.
func yaml_parser_unfold_comments(parser *yaml_parser_t, token *yaml_token_t) {
	// Track if we're adding a head comment with blank lines
	var head_comment_blank_lines = 0

	for parser.comments_head < len(parser.comments) && token.start_mark.index >= parser.comments[parser.comments_head].token_mark.index {
		comment := &parser.comments[parser.comments_head]
		if len(comment.head) > 0 {
			if token.typ == yaml_BLOCK_END_TOKEN {
				// No heads on ends, so keep comment.head for a follow up token.
				break
			}
			if len(parser.head_comment) > 0 {
				parser.head_comment = append(parser.head_comment, '\n')
			}
			parser.head_comment = append(parser.head_comment, comment.head...)
			// Track blank lines from comment
			if parser.preserve_blank_lines && comment.blank_lines_before > 0 {
				// fmt.Printf("DEBUG unfold: Found blank_lines_before=%d from comment with head=%q\n", comment.blank_lines_before, string(comment.head))
				if head_comment_blank_lines == 0 {
					head_comment_blank_lines = comment.blank_lines_before
				}
				// Store directly on parser for head comment
				if parser.head_comment_blank_lines == 0 {
					parser.head_comment_blank_lines = comment.blank_lines_before
				}
			}
		}
		if len(comment.foot) > 0 {
			if len(parser.foot_comment) > 0 {
				parser.foot_comment = append(parser.foot_comment, '\n')
			}
			parser.foot_comment = append(parser.foot_comment, comment.foot...)
		}
		if len(comment.line) > 0 {
			if len(parser.line_comment) > 0 {
				parser.line_comment = append(parser.line_comment, '\n')
			}
			parser.line_comment = append(parser.line_comment, comment.line...)
		}
		*comment = yaml_comment_t{}
		parser.comments_head++
	}

	// If we collected head comment with blank lines, store them in parser state
	// Also store them on the token itself if it's a scalar
	if head_comment_blank_lines > 0 && len(parser.head_comment) > 0 {
		// Store on the token if it's a scalar token
		if token.typ == yaml_SCALAR_TOKEN && token.blank_lines_before == 0 {
			token.blank_lines_before = head_comment_blank_lines
			// fmt.Printf("DEBUG unfold: Set token.blank_lines_before=%d for scalar token\n", head_comment_blank_lines)
		}
		// Also set parser state
		if parser.blank_lines_before == 0 {
			parser.blank_lines_before = head_comment_blank_lines
			// fmt.Printf("DEBUG unfold: Set parser.blank_lines_before=%d for head comment\n", head_comment_blank_lines)
		}
	}
}

// Remove the next token from the queue (must be called after peek_token).
func skip_token(parser *yaml_parser_t) {
	parser.token_available = false
	parser.tokens_parsed++
	parser.stream_end_produced = parser.tokens[parser.tokens_head].typ == yaml_STREAM_END_TOKEN
	parser.tokens_head++
}

// Get the next event.
func yaml_parser_parse(parser *yaml_parser_t, event *yaml_event_t) bool {
	// Erase the event object.
	*event = yaml_event_t{}

	// No events after the end of the stream or error.
	if parser.stream_end_produced || parser.error != yaml_NO_ERROR || parser.state == yaml_PARSE_END_STATE {
		return true
	}

	// Generate the next event.
	return yaml_parser_state_machine(parser, event)
}

// Set parser error.
func yaml_parser_set_parser_error(parser *yaml_parser_t, problem string, problem_mark yaml_mark_t) bool {
	parser.error = yaml_PARSER_ERROR
	parser.problem = problem
	parser.problem_mark = problem_mark
	return false
}

func yaml_parser_set_parser_error_context(parser *yaml_parser_t, context string, context_mark yaml_mark_t, problem string, problem_mark yaml_mark_t) bool {
	parser.error = yaml_PARSER_ERROR
	parser.context = context
	parser.context_mark = context_mark
	parser.problem = problem
	parser.problem_mark = problem_mark
	return false
}

// State dispatcher.
func yaml_parser_state_machine(parser *yaml_parser_t, event *yaml_event_t) bool {
	//trace("yaml_parser_state_machine", "state:", parser.state.String())

	switch parser.state {
	case yaml_PARSE_STREAM_START_STATE:
		return yaml_parser_parse_stream_start(parser, event)

	case yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE:
		return yaml_parser_parse_document_start(parser, event, true)

	case yaml_PARSE_DOCUMENT_START_STATE:
		return yaml_parser_parse_document_start(parser, event, false)

	case yaml_PARSE_DOCUMENT_CONTENT_STATE:
		return yaml_parser_parse_document_content(parser, event)

	case yaml_PARSE_DOCUMENT_END_STATE:
		return yaml_parser_parse_document_end(parser, event)

	case yaml_PARSE_BLOCK_NODE_STATE:
		return yaml_parser_parse_node(parser, event, true, false)

	case yaml_PARSE_BLOCK_NODE_OR_INDENTLESS_SEQUENCE_STATE:
		return yaml_parser_parse_node(parser, event, true, true)

	case yaml_PARSE_FLOW_NODE_STATE:
		return yaml_parser_parse_node(parser, event, false, false)

	case yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE:
		return yaml_parser_parse_block_sequence_entry(parser, event, true)

	case yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_block_sequence_entry(parser, event, false)

	case yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_indentless_sequence_entry(parser, event)

	case yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE:
		return yaml_parser_parse_block_mapping_key(parser, event, true)

	case yaml_PARSE_BLOCK_MAPPING_KEY_STATE:
		return yaml_parser_parse_block_mapping_key(parser, event, false)

	case yaml_PARSE_BLOCK_MAPPING_VALUE_STATE:
		return yaml_parser_parse_block_mapping_value(parser, event)

	case yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE:
		return yaml_parser_parse_flow_sequence_entry(parser, event, true)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE:
		return yaml_parser_parse_flow_sequence_entry(parser, event, false)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_key(parser, event)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_value(parser, event)

	case yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE:
		return yaml_parser_parse_flow_sequence_entry_mapping_end(parser, event)

	case yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE:
		return yaml_parser_parse_flow_mapping_key(parser, event, true)

	case yaml_PARSE_FLOW_MAPPING_KEY_STATE:
		return yaml_parser_parse_flow_mapping_key(parser, event, false)

	case yaml_PARSE_FLOW_MAPPING_VALUE_STATE:
		return yaml_parser_parse_flow_mapping_value(parser, event, false)

	case yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE:
		return yaml_parser_parse_flow_mapping_value(parser, event, true)

	default:
		panic("invalid parser state")
	}
}

// Parse the production:
// stream   ::= STREAM-START implicit_document? explicit_document* STREAM-END
//              ************
func yaml_parser_parse_stream_start(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != yaml_STREAM_START_TOKEN {
		return yaml_parser_set_parser_error(parser, "did not find expected <stream-start>", token.start_mark)
	}
	parser.state = yaml_PARSE_IMPLICIT_DOCUMENT_START_STATE
	*event = yaml_event_t{
		typ:        yaml_STREAM_START_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
		encoding:   token.encoding,
	}
	skip_token(parser)
	return true
}

// Parse the productions:
// implicit_document    ::= block_node DOCUMENT-END*
//                          *
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//                          *************************
func yaml_parser_parse_document_start(parser *yaml_parser_t, event *yaml_event_t, implicit bool) bool {

	token := peek_token(parser)
	if token == nil {
		return false
	}

	// Parse extra document end indicators.
	if !implicit {
		for token.typ == yaml_DOCUMENT_END_TOKEN {
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
		}
	}

	if implicit && token.typ != yaml_VERSION_DIRECTIVE_TOKEN &&
		token.typ != yaml_TAG_DIRECTIVE_TOKEN &&
		token.typ != yaml_DOCUMENT_START_TOKEN &&
		token.typ != yaml_STREAM_END_TOKEN {
		// Parse an implicit document.
		if !yaml_parser_process_directives(parser, nil, nil) {
			return false
		}
		parser.states = append(parser.states, yaml_PARSE_DOCUMENT_END_STATE)
		parser.state = yaml_PARSE_BLOCK_NODE_STATE

		var head_comment []byte
		if len(parser.head_comment) > 0 {
			// [Go] Scan the header comment backwards, and if an empty line is found, break
			//      the header so the part before the last empty line goes into the
			//      document header, while the bottom of it goes into a follow up event.
			for i := len(parser.head_comment) - 1; i > 0; i-- {
				if parser.head_comment[i] == '\n' {
					if i == len(parser.head_comment)-1 {
						head_comment = parser.head_comment[:i]
						parser.head_comment = parser.head_comment[i+1:]
						break
					} else if parser.head_comment[i-1] == '\n' {
						head_comment = parser.head_comment[:i-1]
						parser.head_comment = parser.head_comment[i+1:]
						break
					}
				}
			}
		}

		*event = yaml_event_t{
			typ:        yaml_DOCUMENT_START_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,

			head_comment: head_comment,
		}

	} else if token.typ != yaml_STREAM_END_TOKEN {
		// Parse an explicit document.
		var version_directive *yaml_version_directive_t
		var tag_directives []yaml_tag_directive_t
		start_mark := token.start_mark
		if !yaml_parser_process_directives(parser, &version_directive, &tag_directives) {
			return false
		}
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_DOCUMENT_START_TOKEN {
			yaml_parser_set_parser_error(parser,
				"did not find expected <document start>", token.start_mark)
			return false
		}
		parser.states = append(parser.states, yaml_PARSE_DOCUMENT_END_STATE)
		parser.state = yaml_PARSE_DOCUMENT_CONTENT_STATE
		end_mark := token.end_mark

		*event = yaml_event_t{
			typ:               yaml_DOCUMENT_START_EVENT,
			start_mark:        start_mark,
			end_mark:          end_mark,
			version_directive: version_directive,
			tag_directives:    tag_directives,
			implicit:          false,
		}
		skip_token(parser)

	} else {
		// Parse the stream end.
		parser.state = yaml_PARSE_END_STATE
		*event = yaml_event_t{
			typ:        yaml_STREAM_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		skip_token(parser)
	}

	return true
}

// Parse the productions:
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//                                                    ***********
//
func yaml_parser_parse_document_content(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ == yaml_VERSION_DIRECTIVE_TOKEN ||
		token.typ == yaml_TAG_DIRECTIVE_TOKEN ||
		token.typ == yaml_DOCUMENT_START_TOKEN ||
		token.typ == yaml_DOCUMENT_END_TOKEN ||
		token.typ == yaml_STREAM_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		return yaml_parser_process_empty_scalar(parser, event,
			token.start_mark)
	}
	return yaml_parser_parse_node(parser, event, true, false)
}

// Parse the productions:
// implicit_document    ::= block_node DOCUMENT-END*
//                                     *************
// explicit_document    ::= DIRECTIVE* DOCUMENT-START block_node? DOCUMENT-END*
//
func yaml_parser_parse_document_end(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	start_mark := token.start_mark
	end_mark := token.start_mark

	implicit := true
	if token.typ == yaml_DOCUMENT_END_TOKEN {
		end_mark = token.end_mark
		skip_token(parser)
		implicit = false
	}

	parser.tag_directives = parser.tag_directives[:0]

	parser.state = yaml_PARSE_DOCUMENT_START_STATE
	*event = yaml_event_t{
		typ:        yaml_DOCUMENT_END_EVENT,
		start_mark: start_mark,
		end_mark:   end_mark,
		implicit:   implicit,
	}
	yaml_parser_set_event_comments(parser, event)
	if len(event.head_comment) > 0 && len(event.foot_comment) == 0 {
		event.foot_comment = event.head_comment
		event.head_comment = nil
	}
	return true
}

func yaml_parser_set_event_comments(parser *yaml_parser_t, event *yaml_event_t) {
	event.head_comment = parser.head_comment
	event.line_comment = parser.line_comment
	event.foot_comment = parser.foot_comment
	// If we have a head comment with blank lines, ensure they're set on the event
	if parser.preserve_blank_lines && len(event.head_comment) > 0 {
		// Use the dedicated head_comment_blank_lines field
		if parser.head_comment_blank_lines > 0 {
			// fmt.Printf("DEBUG set_event_comments: Using head_comment_blank_lines=%d\n", parser.head_comment_blank_lines)
			if event.blank_lines_before == 0 {
				event.blank_lines_before = parser.head_comment_blank_lines
			}
			parser.head_comment_blank_lines = 0 // Reset after use
		} else if parser.blank_lines_before > 0 && event.blank_lines_before == 0 {
			// Fallback to parser.blank_lines_before if available
			// fmt.Printf("DEBUG set_event_comments: Using parser.blank_lines_before=%d\n", parser.blank_lines_before)
			event.blank_lines_before = parser.blank_lines_before
		}
	}
	// Always reset parser.blank_lines_before after transferring to event
	parser.blank_lines_before = 0
	parser.head_comment = nil
	parser.line_comment = nil
	parser.foot_comment = nil
	parser.tail_comment = nil
	parser.stem_comment = nil
	parser.head_comment_blank_lines = 0
}

// Parse the productions:
// block_node_or_indentless_sequence    ::=
//                          ALIAS
//                          *****
//                          | properties (block_content | indentless_block_sequence)?
//                            **********  *
//                          | block_content | indentless_block_sequence
//                            *
// block_node           ::= ALIAS
//                          *****
//                          | properties block_content?
//                            ********** *
//                          | block_content
//                            *
// flow_node            ::= ALIAS
//                          *****
//                          | properties flow_content?
//                            ********** *
//                          | flow_content
//                            *
// properties           ::= TAG ANCHOR? | ANCHOR TAG?
//                          *************************
// block_content        ::= block_collection | flow_collection | SCALAR
//                                                               ******
// flow_content         ::= flow_collection | SCALAR
//                                            ******
func yaml_parser_parse_node(parser *yaml_parser_t, event *yaml_event_t, block, indentless_sequence bool) bool {
	//defer trace("yaml_parser_parse_node", "block:", block, "indentless_sequence:", indentless_sequence)()
	// fmt.Printf("DEBUG parser: parse_node called, parser.blank_lines_before=%d\n", parser.blank_lines_before)

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ == yaml_ALIAS_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		*event = yaml_event_t{
			typ:        yaml_ALIAS_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
			anchor:     token.value,
		}
		yaml_parser_set_event_comments(parser, event)
		skip_token(parser)
		return true
	}

	start_mark := token.start_mark
	end_mark := token.start_mark

	var tag_token bool
	var tag_handle, tag_suffix, anchor []byte
	var tag_mark yaml_mark_t
	if token.typ == yaml_ANCHOR_TOKEN {
		anchor = token.value
		start_mark = token.start_mark
		end_mark = token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ == yaml_TAG_TOKEN {
			tag_token = true
			tag_handle = token.value
			tag_suffix = token.suffix
			tag_mark = token.start_mark
			end_mark = token.end_mark
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
		}
	} else if token.typ == yaml_TAG_TOKEN {
		tag_token = true
		tag_handle = token.value
		tag_suffix = token.suffix
		start_mark = token.start_mark
		tag_mark = token.start_mark
		end_mark = token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ == yaml_ANCHOR_TOKEN {
			anchor = token.value
			end_mark = token.end_mark
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
		}
	}

	var tag []byte
	if tag_token {
		if len(tag_handle) == 0 {
			tag = tag_suffix
			tag_suffix = nil
		} else {
			for i := range parser.tag_directives {
				if bytes.Equal(parser.tag_directives[i].handle, tag_handle) {
					tag = append([]byte(nil), parser.tag_directives[i].prefix...)
					tag = append(tag, tag_suffix...)
					break
				}
			}
			if len(tag) == 0 {
				yaml_parser_set_parser_error_context(parser,
					"while parsing a node", start_mark,
					"found undefined tag handle", tag_mark)
				return false
			}
		}
	}

	implicit := len(tag) == 0
	if indentless_sequence && token.typ == yaml_BLOCK_ENTRY_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE
		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_BLOCK_SEQUENCE_STYLE),
			blank_lines_before: parser.blank_lines_before,
			blank_lines_after:  parser.blank_lines_after,
		}
		return true
	}
	if token.typ == yaml_SCALAR_TOKEN {
		var plain_implicit, quoted_implicit bool
		end_mark = token.end_mark
		if (len(tag) == 0 && token.style == yaml_PLAIN_SCALAR_STYLE) || (len(tag) == 1 && tag[0] == '!') {
			plain_implicit = true
		} else if len(tag) == 0 {
			quoted_implicit = true
		}
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = yaml_event_t{
			typ:             yaml_SCALAR_EVENT,
			start_mark:      start_mark,
			end_mark:        end_mark,
			anchor:          anchor,
			tag:             tag,
			value:           token.value,
			implicit:        plain_implicit,
			quoted_implicit: quoted_implicit,
			style:           yaml_style_t(token.style),
			blank_lines_before: token.blank_lines_before,
			blank_lines_after:  0,
		}
		// Use parser's blank lines if they were set (e.g., from BLOCK_ENTRY_TOKEN)
		// Don't consume them yet if we might have a head comment
		if parser.blank_lines_before > 0 {
			// fmt.Printf("DEBUG parser_parse_node: parser.blank_lines_before=%d for scalar %q\n",
				// parser.blank_lines_before, string(token.value))
			if event.blank_lines_before == 0 {
				event.blank_lines_before = parser.blank_lines_before
			}
			// Don't reset parser.blank_lines_before yet - let yaml_parser_set_event_comments handle it
		}
		yaml_parser_set_event_comments(parser, event)
		skip_token(parser)
		return true
	}
	if token.typ == yaml_FLOW_SEQUENCE_START_TOKEN {
		// [Go] Some of the events below can be merged as they differ only on style.
		end_mark = token.end_mark
		parser.state = yaml_PARSE_FLOW_SEQUENCE_FIRST_ENTRY_STATE
		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_FLOW_SEQUENCE_STYLE),
			blank_lines_before: parser.blank_lines_before,
			blank_lines_after:  parser.blank_lines_after,
		}
		yaml_parser_set_event_comments(parser, event)
		return true
	}
	if token.typ == yaml_FLOW_MAPPING_START_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_FLOW_MAPPING_FIRST_KEY_STATE
		*event = yaml_event_t{
			typ:        yaml_MAPPING_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_FLOW_MAPPING_STYLE),
			blank_lines_before: parser.blank_lines_before,
			blank_lines_after:  parser.blank_lines_after,
		}
		yaml_parser_set_event_comments(parser, event)
		return true
	}
	if block && token.typ == yaml_BLOCK_SEQUENCE_START_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_BLOCK_SEQUENCE_FIRST_ENTRY_STATE
		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_BLOCK_SEQUENCE_STYLE),
			blank_lines_before: parser.blank_lines_before,
			blank_lines_after:  parser.blank_lines_after,
		}
		if parser.stem_comment != nil {
			event.head_comment = parser.stem_comment
			parser.stem_comment = nil
		}
		return true
	}
	if block && token.typ == yaml_BLOCK_MAPPING_START_TOKEN {
		end_mark = token.end_mark
		parser.state = yaml_PARSE_BLOCK_MAPPING_FIRST_KEY_STATE
		*event = yaml_event_t{
			typ:        yaml_MAPPING_START_EVENT,
			start_mark: start_mark,
			end_mark:   end_mark,
			anchor:     anchor,
			tag:        tag,
			implicit:   implicit,
			style:      yaml_style_t(yaml_BLOCK_MAPPING_STYLE),
			blank_lines_before: parser.blank_lines_before,
			blank_lines_after:  parser.blank_lines_after,
		}
		if parser.stem_comment != nil {
			event.head_comment = parser.stem_comment
			parser.stem_comment = nil
		}
		return true
	}
	if len(anchor) > 0 || len(tag) > 0 {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]

		*event = yaml_event_t{
			typ:             yaml_SCALAR_EVENT,
			start_mark:      start_mark,
			end_mark:        end_mark,
			anchor:          anchor,
			tag:             tag,
			implicit:        implicit,
			quoted_implicit: false,
			style:           yaml_style_t(yaml_PLAIN_SCALAR_STYLE),
			blank_lines_before: 0,
			blank_lines_after:  0,
		}
		return true
	}

	context := "while parsing a flow node"
	if block {
		context = "while parsing a block node"
	}
	yaml_parser_set_parser_error_context(parser, context, start_mark,
		"did not find expected node content", token.start_mark)
	return false
}

// Parse the productions:
// block_sequence ::= BLOCK-SEQUENCE-START (BLOCK-ENTRY block_node?)* BLOCK-END
//                    ********************  *********** *             *********
//
func yaml_parser_parse_block_sequence_entry(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		if token == nil {
			return false
		}
		// fmt.Printf("DEBUG parser: block_seq first=true, skipping token type=%v\n", token.typ)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}
	// fmt.Printf("DEBUG parser: After skip/start, token type=%v\n", token.typ)

	if token.typ == yaml_BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		prior_head_len := len(parser.head_comment)
		// Save blank lines from the block entry token
		blank_lines := token.blank_lines_before
		// fmt.Printf("DEBUG parser: BLOCK_ENTRY found at line %d, blank_lines=%d\n",
		//	token.start_mark.line, blank_lines)
		skip_token(parser)
		// Transfer blank lines to parser for the next node
		// Always set it, even if 0, to ensure clean state
		parser.blank_lines_before = blank_lines
		if blank_lines > 0 {
			// fmt.Printf("DEBUG parser[1]: BLOCK_ENTRY transferred blank_lines=%d\n", blank_lines)
		}
		yaml_parser_split_stem_comment(parser, prior_head_len)
		// Save blank lines before peeking (which might reset them)
		saved_blank_lines := parser.blank_lines_before
		token = peek_token(parser)
		if token == nil {
			return false
		}
		// Restore blank lines after peeking
		parser.blank_lines_before = saved_blank_lines
		if token.typ != yaml_BLOCK_ENTRY_TOKEN && token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE)
			// fmt.Printf("DEBUG parser[2]: About to call parse_node, blank_lines=%d\n", parser.blank_lines_before)
			result := yaml_parser_parse_node(parser, event, true, false)
			// fmt.Printf("DEBUG parser: After parse_node, result=%v\n", result)
			return result
		} else {
			parser.state = yaml_PARSE_BLOCK_SEQUENCE_ENTRY_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	}
	if token.typ == yaml_BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]

		*event = yaml_event_t{
			typ:        yaml_SEQUENCE_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}

		skip_token(parser)
		return true
	}

	context_mark := parser.marks[len(parser.marks)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	return yaml_parser_set_parser_error_context(parser,
		"while parsing a block collection", context_mark,
		"did not find expected '-' indicator", token.start_mark)
}

// Parse the productions:
// indentless_sequence  ::= (BLOCK-ENTRY block_node?)+
//                           *********** *
func yaml_parser_parse_indentless_sequence_entry(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ == yaml_BLOCK_ENTRY_TOKEN {
		mark := token.end_mark
		prior_head_len := len(parser.head_comment)
		// Save blank lines from the block entry token
		blank_lines := token.blank_lines_before
		skip_token(parser)
		// Transfer blank lines to parser for the next node
		if blank_lines > 0 {
			// fmt.Printf("DEBUG parser: BLOCK_ENTRY (indentless) transferring blank_lines=%d to parser\n", blank_lines)
			parser.blank_lines_before = blank_lines
		}
		yaml_parser_split_stem_comment(parser, prior_head_len)
		// Save blank lines before peeking (which might reset them)
		saved_blank_lines := parser.blank_lines_before
		token = peek_token(parser)
		if token == nil {
			return false
		}
		// Restore blank lines after peeking
		parser.blank_lines_before = saved_blank_lines
		if token.typ != yaml_BLOCK_ENTRY_TOKEN &&
			token.typ != yaml_KEY_TOKEN &&
			token.typ != yaml_VALUE_TOKEN &&
			token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, true, false)
		}
		parser.state = yaml_PARSE_INDENTLESS_SEQUENCE_ENTRY_STATE
		return yaml_parser_process_empty_scalar(parser, event, mark)
	}
	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]

	*event = yaml_event_t{
		typ:        yaml_SEQUENCE_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.start_mark, // [Go] Shouldn't this be token.end_mark?
	}
	return true
}

// Split stem comment from head comment.
//
// When a sequence or map is found under a sequence entry, the former head comment
// is assigned to the underlying sequence or map as a whole, not the individual
// sequence or map entry as would be expected otherwise. To handle this case the
// previous head comment is moved aside as the stem comment.
func yaml_parser_split_stem_comment(parser *yaml_parser_t, stem_len int) {
	if stem_len == 0 {
		return
	}

	token := peek_token(parser)
	if token == nil || token.typ != yaml_BLOCK_SEQUENCE_START_TOKEN && token.typ != yaml_BLOCK_MAPPING_START_TOKEN {
		return
	}

	parser.stem_comment = parser.head_comment[:stem_len]
	if len(parser.head_comment) == stem_len {
		parser.head_comment = nil
	} else {
		// Copy suffix to prevent very strange bugs if someone ever appends
		// further bytes to the prefix in the stem_comment slice above.
		parser.head_comment = append([]byte(nil), parser.head_comment[stem_len+1:]...)
	}
}

// Parse the productions:
// block_mapping        ::= BLOCK-MAPPING_START
//                          *******************
//                          ((KEY block_node_or_indentless_sequence?)?
//                            *** *
//                          (VALUE block_node_or_indentless_sequence?)?)*
//
//                          BLOCK-END
//                          *********
//
func yaml_parser_parse_block_mapping_key(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		if token == nil {
			return false
		}
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	// [Go] A tail comment was left from the prior mapping value processed. Emit an event
	//      as it needs to be processed with that value and not the following key.
	if len(parser.tail_comment) > 0 {
		*event = yaml_event_t{
			typ:          yaml_TAIL_COMMENT_EVENT,
			start_mark:   token.start_mark,
			end_mark:     token.end_mark,
			foot_comment: parser.tail_comment,
		}
		parser.tail_comment = nil
		return true
	}

	if token.typ == yaml_KEY_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_KEY_TOKEN &&
			token.typ != yaml_VALUE_TOKEN &&
			token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_BLOCK_MAPPING_VALUE_STATE)
			return yaml_parser_parse_node(parser, event, true, true)
		} else {
			parser.state = yaml_PARSE_BLOCK_MAPPING_VALUE_STATE
			return yaml_parser_process_empty_scalar(parser, event, mark)
		}
	} else if token.typ == yaml_BLOCK_END_TOKEN {
		parser.state = parser.states[len(parser.states)-1]
		parser.states = parser.states[:len(parser.states)-1]
		parser.marks = parser.marks[:len(parser.marks)-1]
		*event = yaml_event_t{
			typ:        yaml_MAPPING_END_EVENT,
			start_mark: token.start_mark,
			end_mark:   token.end_mark,
		}
		yaml_parser_set_event_comments(parser, event)
		skip_token(parser)
		return true
	}

	context_mark := parser.marks[len(parser.marks)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	return yaml_parser_set_parser_error_context(parser,
		"while parsing a block mapping", context_mark,
		"did not find expected key", token.start_mark)
}

// Parse the productions:
// block_mapping        ::= BLOCK-MAPPING_START
//
//                          ((KEY block_node_or_indentless_sequence?)?
//
//                          (VALUE block_node_or_indentless_sequence?)?)*
//                           ***** *
//                          BLOCK-END
//
//
func yaml_parser_parse_block_mapping_value(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ == yaml_VALUE_TOKEN {
		mark := token.end_mark
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_KEY_TOKEN &&
			token.typ != yaml_VALUE_TOKEN &&
			token.typ != yaml_BLOCK_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_BLOCK_MAPPING_KEY_STATE)
			return yaml_parser_parse_node(parser, event, true, true)
		}
		parser.state = yaml_PARSE_BLOCK_MAPPING_KEY_STATE
		return yaml_parser_process_empty_scalar(parser, event, mark)
	}
	parser.state = yaml_PARSE_BLOCK_MAPPING_KEY_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

// Parse the productions:
// flow_sequence        ::= FLOW-SEQUENCE-START
//                          *******************
//                          (flow_sequence_entry FLOW-ENTRY)*
//                           *                   **********
//                          flow_sequence_entry?
//                          *
//                          FLOW-SEQUENCE-END
//                          *****************
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                          *
//
func yaml_parser_parse_flow_sequence_entry(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		if token == nil {
			return false
		}
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
		if !first {
			if token.typ == yaml_FLOW_ENTRY_TOKEN {
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}
			} else {
				context_mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]
				return yaml_parser_set_parser_error_context(parser,
					"while parsing a flow sequence", context_mark,
					"did not find expected ',' or ']'", token.start_mark)
			}
		}

		if token.typ == yaml_KEY_TOKEN {
			parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_KEY_STATE
			*event = yaml_event_t{
				typ:        yaml_MAPPING_START_EVENT,
				start_mark: token.start_mark,
				end_mark:   token.end_mark,
				implicit:   true,
				style:      yaml_style_t(yaml_FLOW_MAPPING_STYLE),
				blank_lines_before: parser.blank_lines_before,
				blank_lines_after:  parser.blank_lines_after,
			}
			skip_token(parser)
			return true
		} else if token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]

	*event = yaml_event_t{
		typ:        yaml_SEQUENCE_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}
	yaml_parser_set_event_comments(parser, event)

	skip_token(parser)
	return true
}

//
// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                      *** *
//
func yaml_parser_parse_flow_sequence_entry_mapping_key(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ != yaml_VALUE_TOKEN &&
		token.typ != yaml_FLOW_ENTRY_TOKEN &&
		token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
		parser.states = append(parser.states, yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE)
		return yaml_parser_parse_node(parser, event, false, false)
	}
	mark := token.end_mark
	skip_token(parser)
	parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_VALUE_STATE
	return yaml_parser_process_empty_scalar(parser, event, mark)
}

// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                                      ***** *
//
func yaml_parser_parse_flow_sequence_entry_mapping_value(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if token.typ == yaml_VALUE_TOKEN {
		skip_token(parser)
		token := peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_FLOW_ENTRY_TOKEN && token.typ != yaml_FLOW_SEQUENCE_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}
	parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_MAPPING_END_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

// Parse the productions:
// flow_sequence_entry  ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                                                      *
//
func yaml_parser_parse_flow_sequence_entry_mapping_end(parser *yaml_parser_t, event *yaml_event_t) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	parser.state = yaml_PARSE_FLOW_SEQUENCE_ENTRY_STATE
	*event = yaml_event_t{
		typ:        yaml_MAPPING_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.start_mark, // [Go] Shouldn't this be end_mark?
	}
	return true
}

// Parse the productions:
// flow_mapping         ::= FLOW-MAPPING-START
//                          ******************
//                          (flow_mapping_entry FLOW-ENTRY)*
//                           *                  **********
//                          flow_mapping_entry?
//                          ******************
//                          FLOW-MAPPING-END
//                          ****************
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                          *           *** *
//
func yaml_parser_parse_flow_mapping_key(parser *yaml_parser_t, event *yaml_event_t, first bool) bool {
	if first {
		token := peek_token(parser)
		parser.marks = append(parser.marks, token.start_mark)
		skip_token(parser)
	}

	token := peek_token(parser)
	if token == nil {
		return false
	}

	if token.typ != yaml_FLOW_MAPPING_END_TOKEN {
		if !first {
			if token.typ == yaml_FLOW_ENTRY_TOKEN {
				skip_token(parser)
				token = peek_token(parser)
				if token == nil {
					return false
				}
			} else {
				context_mark := parser.marks[len(parser.marks)-1]
				parser.marks = parser.marks[:len(parser.marks)-1]
				return yaml_parser_set_parser_error_context(parser,
					"while parsing a flow mapping", context_mark,
					"did not find expected ',' or '}'", token.start_mark)
			}
		}

		if token.typ == yaml_KEY_TOKEN {
			skip_token(parser)
			token = peek_token(parser)
			if token == nil {
				return false
			}
			if token.typ != yaml_VALUE_TOKEN &&
				token.typ != yaml_FLOW_ENTRY_TOKEN &&
				token.typ != yaml_FLOW_MAPPING_END_TOKEN {
				parser.states = append(parser.states, yaml_PARSE_FLOW_MAPPING_VALUE_STATE)
				return yaml_parser_parse_node(parser, event, false, false)
			} else {
				parser.state = yaml_PARSE_FLOW_MAPPING_VALUE_STATE
				return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
			}
		} else if token.typ != yaml_FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_MAPPING_EMPTY_VALUE_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}

	parser.state = parser.states[len(parser.states)-1]
	parser.states = parser.states[:len(parser.states)-1]
	parser.marks = parser.marks[:len(parser.marks)-1]
	*event = yaml_event_t{
		typ:        yaml_MAPPING_END_EVENT,
		start_mark: token.start_mark,
		end_mark:   token.end_mark,
	}
	yaml_parser_set_event_comments(parser, event)
	skip_token(parser)
	return true
}

// Parse the productions:
// flow_mapping_entry   ::= flow_node | KEY flow_node? (VALUE flow_node?)?
//                                   *                  ***** *
//
func yaml_parser_parse_flow_mapping_value(parser *yaml_parser_t, event *yaml_event_t, empty bool) bool {
	token := peek_token(parser)
	if token == nil {
		return false
	}
	if empty {
		parser.state = yaml_PARSE_FLOW_MAPPING_KEY_STATE
		return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
	}
	if token.typ == yaml_VALUE_TOKEN {
		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
		if token.typ != yaml_FLOW_ENTRY_TOKEN && token.typ != yaml_FLOW_MAPPING_END_TOKEN {
			parser.states = append(parser.states, yaml_PARSE_FLOW_MAPPING_KEY_STATE)
			return yaml_parser_parse_node(parser, event, false, false)
		}
	}
	parser.state = yaml_PARSE_FLOW_MAPPING_KEY_STATE
	return yaml_parser_process_empty_scalar(parser, event, token.start_mark)
}

// Generate an empty scalar event.
func yaml_parser_process_empty_scalar(parser *yaml_parser_t, event *yaml_event_t, mark yaml_mark_t) bool {
	*event = yaml_event_t{
		typ:        yaml_SCALAR_EVENT,
		start_mark: mark,
		end_mark:   mark,
		value:      nil, // Empty
		implicit:   true,
		style:      yaml_style_t(yaml_PLAIN_SCALAR_STYLE),
		blank_lines_before: 0,
		blank_lines_after:  0,
	}
	return true
}

var default_tag_directives = []yaml_tag_directive_t{
	{[]byte("!"), []byte("!")},
	{[]byte("!!"), []byte("tag:yaml.org,2002:")},
}

// Parse directives.
func yaml_parser_process_directives(parser *yaml_parser_t,
	version_directive_ref **yaml_version_directive_t,
	tag_directives_ref *[]yaml_tag_directive_t) bool {

	var version_directive *yaml_version_directive_t
	var tag_directives []yaml_tag_directive_t

	token := peek_token(parser)
	if token == nil {
		return false
	}

	for token.typ == yaml_VERSION_DIRECTIVE_TOKEN || token.typ == yaml_TAG_DIRECTIVE_TOKEN {
		if token.typ == yaml_VERSION_DIRECTIVE_TOKEN {
			if version_directive != nil {
				yaml_parser_set_parser_error(parser,
					"found duplicate %YAML directive", token.start_mark)
				return false
			}
			if token.major != 1 || token.minor != 1 {
				yaml_parser_set_parser_error(parser,
					"found incompatible YAML document", token.start_mark)
				return false
			}
			version_directive = &yaml_version_directive_t{
				major: token.major,
				minor: token.minor,
			}
		} else if token.typ == yaml_TAG_DIRECTIVE_TOKEN {
			value := yaml_tag_directive_t{
				handle: token.value,
				prefix: token.prefix,
			}
			if !yaml_parser_append_tag_directive(parser, value, false, token.start_mark) {
				return false
			}
			tag_directives = append(tag_directives, value)
		}

		skip_token(parser)
		token = peek_token(parser)
		if token == nil {
			return false
		}
	}

	for i := range default_tag_directives {
		if !yaml_parser_append_tag_directive(parser, default_tag_directives[i], true, token.start_mark) {
			return false
		}
	}

	if version_directive_ref != nil {
		*version_directive_ref = version_directive
	}
	if tag_directives_ref != nil {
		*tag_directives_ref = tag_directives
	}
	return true
}

// Append a tag directive to the directives stack.
func yaml_parser_append_tag_directive(parser *yaml_parser_t, value yaml_tag_directive_t, allow_duplicates bool, mark yaml_mark_t) bool {
	for i := range parser.tag_directives {
		if bytes.Equal(value.handle, parser.tag_directives[i].handle) {
			if allow_duplicates {
				return true
			}
			return yaml_parser_set_parser_error(parser, "found duplicate %TAG directive", mark)
		}
	}

	// [Go] I suspect the copy is unnecessary. This was likely done
	// because there was no way to track ownership of the data.
	value_copy := yaml_tag_directive_t{
		handle: make([]byte, len(value.handle)),
		prefix: make([]byte, len(value.prefix)),
	}
	copy(value_copy.handle, value.handle)
	copy(value_copy.prefix, value.prefix)
	parser.tag_directives = append(parser.tag_directives, value_copy)
	return true
}
