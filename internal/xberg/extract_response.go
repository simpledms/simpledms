package xberg

type extractResponse struct {
	Results []struct {
		Content string `json:"content"`
	} `json:"results"`
	Errors []struct {
		ErrorType string `json:"error_type"`
		Message   string `json:"message"`
	} `json:"errors"`
}
