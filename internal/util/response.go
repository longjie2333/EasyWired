package util

import (
	"encoding/json"
	"net/http"

	"easywired/internal/model"
)

const (
	CodeNodeNotReady          = "NODE_NOT_READY"
	CodeBadRequest            = "BAD_REQUEST"
	CodeNoAddressPool         = "NO_ADDRESS_POOL"
	CodeNoAvailableIP         = "NO_AVAILABLE_IP"
	CodeWGConfigFailed        = "WG_CONFIG_FAILED"
	CodeBackendNotSupported   = "BACKEND_NOT_SUPPORTED"
	CodeBackendNotImplemented = "BACKEND_NOT_IMPLEMENTED"
	CodeStoreFailed           = "STORE_FAILED"
	CodePeerNotFound          = "PEER_NOT_FOUND"
	CodeInternalError         = "INTERNAL_ERROR"
)

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func WriteError(w http.ResponseWriter, status int, code, message string) {
	WriteJSON(w, status, model.ErrorResponse{Code: code, Message: message})
}
