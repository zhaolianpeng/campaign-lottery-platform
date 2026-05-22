package response

import (
	"encoding/json"
	"net/http"
)

type Envelope struct {
	Code    string `json:"code"`
	Message string `json:"message"`
	Data    any    `json:"data,omitempty"`
}

func JSON(writer http.ResponseWriter, status int, code string, message string, data any) {
	writer.Header().Set("Content-Type", "application/json; charset=utf-8")
	writer.WriteHeader(status)
	_ = json.NewEncoder(writer).Encode(Envelope{
		Code:    code,
		Message: message,
		Data:    data,
	})
}
