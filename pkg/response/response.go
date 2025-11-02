package response

import (
	"encoding/json"
	"net/http"
)

type APIResponse struct {
	Status  int         `json:"status"`
	Success bool        `json:"success"`
	Message string      `json:"message,omitempty"`
	Data    interface{} `json:"data,omitempty"`
	Errors  interface{} `json:"errors,omitempty"`
}

func JSON(w http.ResponseWriter, statusCode int, resp APIResponse) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(resp)
}

func Success(w http.ResponseWriter, data interface{}, message string) {

	JSON(w, http.StatusOK, APIResponse{
		Status:  http.StatusOK,
		Success: true,
		Message: message,
		Data:    data,
	})
}

func Error(w http.ResponseWriter, statusCode int, message string, errs interface{}) {
	JSON(w, statusCode, APIResponse{
		Status:  statusCode,
		Success: false,
		Message: message,
		Errors:  errs,
	})
}
