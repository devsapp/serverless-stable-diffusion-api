package module

// Error defines model for Error.
type Error struct {
	// Code Error code
	Code int32 `json:"code"`

	// Message Error message
	Message string `json:"message"`
}
