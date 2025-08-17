package event

// APIError represents the structure of an API error response
type APIError struct {
	Type  string         `json:"type"`
	Error APIErrorDetail `json:"error"`
}

// APIErrorDetail contains the details of an API error
type APIErrorDetail struct {
	Type    string `json:"type"`
	Message string `json:"message"`
}

// APIErrorMessage represents an API error message with status code
type APIErrorMessage struct {
	StatusCode int
	Error      APIError
}
