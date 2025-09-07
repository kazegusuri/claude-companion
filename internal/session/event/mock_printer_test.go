package event

// mockPrinter is a mock implementation of the Printer interface for testing
type mockPrinter struct{}

// Print implements the Printer interface
func (p *mockPrinter) Print(event interface{}) {
	// Do nothing for testing
}
