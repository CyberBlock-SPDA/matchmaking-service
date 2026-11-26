package models

const (
	CodeInvalidRequest   = "INVALID_REQUEST"
	CodeNoServers        = "NO_GAME_SERVERS_AVAILABLE"
	CodeAllocationFailed = "ALLOCATION_FAILED"
	CodeInternalError    = "INTERNAL_ERROR"
)

type Problem struct {
	Type   string `json:"type"`
	Title  string `json:"title"`
	Status int    `json:"status"`
	Code   string `json:"code"`
	Detail string `json:"detail,omitempty"`
}
