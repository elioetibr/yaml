# Feature: Blank Lines Are Lost During YAML Document Save Operations

## Summary
When saving/encoding YAML documents, empty blank lines between elements are being stripped out, causing unnecessary diffs in version-controlled files and losing the original document formatting.

## Problem Description

The YAML encoder/marshaler removes blank lines that exist between mapping entries, sequence items, and document sections. This causes:
- Unnecessary diffs in version control systems
- Loss of visual separation that aids readability
- Inability to preserve original document formatting during round-trip operations

## Root Cause Analysis

The current implementation tracks comments (HeadComment, LineComment, FootComment) but doesn't preserve blank lines between elements. The parser recognizes blank lines as delimiters between head/foot comments but discards the blank line information itself. The emitter has no mechanism to output preserved blank lines between nodes.

Key issues:
1. **Node Structure**: The `Node` struct has no field to track blank lines before/after elements
2. **Parser**: `scannerc.go` and `parserc.go` detect blank lines for comment separation but don't preserve them
3. **Emitter**: `emitterc.go` has no logic to emit blank lines between elements
4. **Comment Handling**: Blank lines are used to separate HeadComment from FootComment but aren't stored

## Code Locations Requiring Updates

### Critical Files to Modify

1. **yaml.go** (Lines ~380-420)
   - Add new fields to `Node` struct:
     - `BlankLinesBefore int` - Number of blank lines before this node
     - `BlankLinesAfter int` - Number of blank lines after this node

2. **parserc.go** (Lines ~730-800)
   - Modify `yaml_parser_parse_node` to track blank lines
   - Update comment parsing logic to count empty lines
   - Store blank line count in Node during parsing

3. **scannerc.go** (Lines ~2800-3000)
   - Update `yaml_parser_scan_comments` to preserve blank line counts
   - Modify comment splitting logic to track empty lines separately

4. **emitterc.go** (Lines ~1650-1750)
   - Update `yaml_emitter_emit_node` to output blank lines
   - Add new function `yaml_emitter_write_blank_lines(count int)`
   - Modify comment emission functions to handle blank lines

5. **encode.go** (Lines ~260-350)
   - Update `nodev` function to pass blank line information to emitter
   - Ensure blank lines are emitted for all node types (Scalar, Mapping, Sequence)

6. **decode.go** (Lines ~190-200)
   - Update `node` function in parser to capture blank line information

## Proposed Solution

### 1. Extend Node Structure
```go
type Node struct {
    // ... existing fields ...

    // Blank line preservation
    BlankLinesBefore int  // Number of blank lines before this node
    BlankLinesAfter  int  // Number of blank lines after this node (for sequences/mappings)

    // ... existing fields ...
}
```

### 2. Parser Modifications
- Track consecutive newlines during scanning
- Distinguish between comment separators and standalone blank lines
- Store blank line counts in the Node during construction

### 3. Emitter Modifications
- Before emitting a node, output `BlankLinesBefore` newlines
- After emitting sequence/mapping items, output `BlankLinesAfter` newlines
- Ensure proper interaction with existing comment emission

### 4. Round-trip Preservation
- Ensure blank lines are preserved during parse → encode → parse cycles
- Handle edge cases: document start/end, nested structures, flow styles

## Test Cases to Add

### Test Case 1

```yaml
# Input YAML with blank lines
key1: value1

key2: value2


key3: value3

list:
  - item1

  - item2

  - item3
```

### Test Case 2

```yaml
# Input YAML with blank lines
key1: value1

# key2 head comment
key2: value2
# -- key2 footer comment

# key3 head comment
key3:
# key3 footer comment
  
  # mapping header comment
  # this has mappings and list
  mapping:
    # a mapping
    a: this is the a key
    
    # b mapping
    b: this is the b key
    
    # listKeys 
    listKeys:
      - a
      - b
      - c


# simple list
list:
  - item1

  # this should have this comment
  - item2

  - item3
```

Should preserve all blank lines after round-trip encoding.

## Impact Analysis

### Breaking Changes
- **None expected** - The changes are additive to the Node struct
- Existing code that doesn't use the new fields will continue to work
- Default behavior (zero values) will not add blank lines, maintaining compatibility

### Performance Considerations
- **Minimal impact** - Two additional int fields per Node (16 bytes on 64-bit systems)
- Parsing: Slight overhead to count blank lines (already scanning these lines for comments)
- Emitting: Negligible cost to output additional newlines
- Memory: For a 1000-node document, ~16KB additional memory

### Compatibility
- **Forward compatible**: Documents with blank lines will parse correctly in older versions (blank lines ignored)
- **Backward compatible**: Documents without blank line metadata will work as before
- **YAML spec compliant**: Blank lines are valid YAML and don't affect semantic meaning

## Related Issues
- Similar issues in other YAML libraries:
  - PyYAML: https://github.com/yaml/pyyaml/issues/46
  - ruamel.yaml specifically added this feature for round-trip preservation
  - js-yaml: https://github.com/nodeca/js-yaml/issues/404

## Priority
**Medium-High** - This feature is critical for:
- DevOps tools that modify YAML configs (Kubernetes, Docker Compose, CI/CD)
- Configuration management systems
- Any tool that performs round-trip YAML editing
- Version control workflows where minimal diffs are important

## Implementation Checklist

### Phase 1: Core Implementation
- [ ] Add `BlankLinesBefore` and `BlankLinesAfter` fields to Node struct
- [ ] Update parser to count and store blank lines
- [ ] Update emitter to output preserved blank lines
- [ ] Ensure IsZero() method accounts for new fields

### Phase 2: Testing
- [ ] Add round-trip tests for blank line preservation
- [ ] Test edge cases (document boundaries, nested structures)
- [ ] Test interaction with comments
- [ ] Benchmark performance impact

### Phase 3: Documentation
- [ ] Update Node struct documentation
- [ ] Add examples showing blank line preservation
- [ ] Document any limitations or edge cases

### Phase 4: Integration
- [ ] Ensure compatibility with existing test suite
- [ ] Update CHANGELOG
- [ ] Consider feature flag for gradual rollout (if needed)

## Version Management

### Recommended Version Strategy

**Use v3.1.0** - This is a feature addition, not a breaking change:

```bash
# After implementation and testing
git tag v3.1.0
git push origin v3.1.0
```

### Why Not v4
- **No Breaking Changes** - Feature is purely additive
- **Backward Compatible** - Existing code works unchanged
- **Go Module Convention** - Major versions (v3→v4) only for breaking changes
- **Import Path Stability** - Avoids forcing users to update imports from `github.com/elioetibr/yaml` to `gopkg.in/yaml.v4`

### Feature Flag Option (Optional)

For safer rollout, consider adding a feature flag:

```go
// Global flag approach
package yaml

var PreserveBlankLines = false // Default off for v3.0.x, can default to true in v3.1.0

// Per-instance approach (preferred)
type Decoder struct {
    // ... existing fields ...
    preserveBlankLines bool
}

func (dec *Decoder) SetPreserveBlankLines(enabled bool) {
    dec.preserveBlankLines = enabled
}

// Same for Encoder
func (enc *Encoder) SetPreserveBlankLines(enabled bool) {
    enc.preserveBlankLines = enabled
}
```

### Version Rollout Plan

1. **v3.0.x** (Optional Patch Release)
   - Feature flag disabled by default
   - Early adopters can opt-in
   - Gather feedback

2. **v3.1.0** (Feature Release)
   - Feature flag enabled by default (or removed entirely)
   - Full documentation
   - Stable implementation

### CHANGELOG Entry

```markdown
## v3.1.0 - [Date]

### Features
- Add blank line preservation for round-trip YAML editing (#[PR])
  - New Node fields: `BlankLinesBefore` and `BlankLinesAfter`
  - Preserves document formatting during parse/encode cycles
  - Maintains visual structure and reduces version control diffs

### Compatibility
- Fully backward compatible
- No breaking changes
- Zero-value behavior maintains existing functionality
- Documents without blank line metadata work as before

### Performance
- Minimal impact: ~16 bytes additional memory per Node
- Negligible parsing/encoding overhead
```