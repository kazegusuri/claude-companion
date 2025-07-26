# TODO List for Claude Companion

## Pending

### Enhance assistant message display to be more companion-like (Medium Priority)
Make the assistant's actions more visible and useful by enhancing the output format:

1. **Extract and highlight code blocks from text content**
   - Parse code blocks within assistant text messages
   - Display with language identifier
   - Example:
   ```
   [15:04:05] ASSISTANT (claude-3-opus):
     Text: Here's the Python function you requested:
     Code Block (python):
       def calculate_sum(a, b):
           return a + b
     Text: This function takes two parameters...
   ```

2. **Show tool execution flow**
   - Display when tools are being executed
   - Show tool results inline
   - Example:
   ```
   [15:04:05] ASSISTANT (claude-3-opus):
     Text: Let me search for that information...
     Tool Use: WebSearch (id: toolu_123) → Executing...
     Tool Result: Found 5 relevant results
     Text: Based on the search results...
   ```

3. **Track file operations specifically**
   - Highlight file reads, writes, and edits
   - Show file paths and operation details
   - Example:
   ```
   [15:04:05] ASSISTANT (claude-3-opus):
     File Read: /path/to/file.go (lines 1-50)
     Text: I found the issue in the code...
     File Edit: /path/to/file.go (line 25: "old" → "new")
   ```

### Implementation Notes
- These enhancements would require parsing the content more deeply
- May need to track state between related events (tool use → tool result)
- Consider making display options configurable via command-line flags