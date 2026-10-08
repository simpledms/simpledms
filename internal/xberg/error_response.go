package xberg

type errorResponse struct {
	ErrorType string `json:"error_type"`
	Message   string `json:"message"`
}
