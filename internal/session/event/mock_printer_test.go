package event

import (
	"fmt"

	internalevent "github.com/kazegusuri/claude-companion/internal/event"
)

// mockPrinter is a mock implementation of the Printer interface for testing
type mockPrinter struct{}

// Print implements the Printer interface
func (p *mockPrinter) Print(event interface{}) {
	// Special handling for TaskCompletionMessage to output expected format
	if taskMsg, ok := event.(*internalevent.TaskCompletionMessage); ok {
		timestamp := taskMsg.Timestamp.Format("15:04:05")
		if taskMsg.TaskInfo.SubagentType != "" {
			fmt.Printf("[%s] 💬 %s agentがタスク「%s」を完了しました\n",
				timestamp,
				taskMsg.TaskInfo.SubagentType,
				taskMsg.TaskInfo.Description)
		} else {
			fmt.Printf("[%s] 💬 タスク「%s」を完了しました\n",
				timestamp,
				taskMsg.TaskInfo.Description)
		}
	}
	// Do nothing for other events in testing
}
