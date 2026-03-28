package ws

import "encoding/json"

type Message struct {
	Type string          `json:"type"`
	Data json.RawMessage `json:"data,omitempty"`
}

type InputData struct {
	Text string `json:"text"`
}

type OutputData struct {
	Text   string `json:"text"`
	Stderr bool   `json:"stderr"`
}

type ErrorData struct {
	Code    string `json:"code"`
	Message string `json:"message"`
}

type ResizeData struct {
	Cols int `json:"cols"`
	Rows int `json:"rows"`
}
