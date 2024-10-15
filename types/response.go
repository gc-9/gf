package types

const (
	StatusCodeSuccess      int = 200
	StatusCodeUnauthorized int = 401
	StatusCodeNoPermission int = 403
	StatusCodeNotFound     int = 404
	StatusCodeError        int = 500
)

type JsonResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data"`
}

var SuccessResponse = &JsonResponse{Code: StatusCodeSuccess, Message: "ok"}
