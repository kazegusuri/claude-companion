package event

import (
	internalevent "github.com/kazegusuri/claude-companion/internal/event"
)

var summaryEventTestCases = []parserIntegrationTestCase{
	{
		name:            "basic",
		input:           `{"type":"summary","summary":"Summary text","leafUuid":"leaf_123"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SummaryEvent{
			Session: internalevent.Session{
				SessionID:      "",
				TranscriptPath: "",
			},
			LeafUUID: "leaf_123",
			Summary:  "Summary text",
		},
	},
	{
		name:            "with_long_text",
		input:           `{"type":"summary","summary":"This is a longer summary text that contains multiple sentences. It describes what happened in the session.","leafUuid":"leaf_xyz"}`,
		expectEventSent: true,
		wantEvent: &internalevent.SummaryEvent{
			Session: internalevent.Session{
				SessionID:      "",
				TranscriptPath: "",
			},
			LeafUUID: "leaf_xyz",
			Summary:  "This is a longer summary text that contains multiple sentences. It describes what happened in the session.",
		},
	},
}