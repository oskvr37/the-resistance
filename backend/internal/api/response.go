package api

import (
	"encoding/json"
	"net/http"
	"resistance/internal/response"
	"resistance/internal/store"
)

func writeResponse[T any](w http.ResponseWriter, res *store.RepositoryResponse[T]) {
	w.Header().Set("Content-Type", "application/json")

	// 1. Look up the status code in your map
	statusCode, exists := response.StatusMap[response.ResponseCode(res.Code)]

	// 2. Default to 200 OK if not found, or 500 if the code is explicitly "INTERNAL_ERROR"
	if !exists || res.Code == "" {
		statusCode = http.StatusOK
	}

	w.WriteHeader(statusCode)
	json.NewEncoder(w).Encode(res)
}

func writeError(w http.ResponseWriter, code response.ResponseCode) {
	writeResponse(w, &store.RepositoryResponse[any]{
		Code: code,
	})
}
