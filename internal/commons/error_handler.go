package commons

import (
	"encoding/json"
	"net/http"
)

type ApiErrors string

const (
	// Predefined Errors
	ERR_INTERNAL_TRYAGAIN ApiErrors = "ERR_INTERNAL_TRYAGAIN"
	ERR_INTERNAL_DB_FAIL            = "ERR_INTERNAL_DB_FAIL"

	ERR_AUTH_REQUIRED      = "ERR_AUTH_REQUIRED"
	ERR_AUTH_DEAD          = "ERR_AUTH_DEAD"
	ERR_AUTH_INVALID       = "ERR_AUTH_INVALID"
	ERR_AUTH_NO_PERMISSION = "ERR_AUTH_NO_PERMISSION"

	ERR_REQ_BODY_EMPTY     = "ERR_REQ_BODY_EMPTY"
	ERR_REQ_BODY_MISSING   = "ERR_REQ_BODY_MISSING"
	ERR_REQ_BODY_INVALID   = "ERR_REQ_BODY_INVALID"
	ERR_REQ_METHOD_INVALID = "ERR_REQ_METHOD_INVALID"
	ERR_REQ_PATH_INVALID   = "ERR_REQ_PATH_INVALID"
)

var apiErrorCodes = map[ApiErrors]int{
	ERR_INTERNAL_TRYAGAIN: 503,
	ERR_INTERNAL_DB_FAIL:  500,

	ERR_AUTH_REQUIRED:      401,
	ERR_AUTH_DEAD:          401,
	ERR_AUTH_INVALID:       403,
	ERR_AUTH_NO_PERMISSION: 403,

	ERR_REQ_BODY_EMPTY:     400,
	ERR_REQ_BODY_MISSING:   400,
	ERR_REQ_BODY_INVALID:   400,
	ERR_REQ_METHOD_INVALID: 403,
	ERR_REQ_PATH_INVALID:   404,
}

type ApiError struct {
	Error     ApiErrors          `json:"errcode"`
	Errorinfo any                `json:"errinfo,omitempty"`
	Data      *map[string]string `json:"data,omitempty"`
}

func (e ApiError) NewApiError(i ApiErrors) ApiError {
	return ApiError{
		Error: i,
	}
}

func (e ApiError) HTTPSend(w http.ResponseWriter) {
	dat, err := json.Marshal(e)
	d := string("")
	if err != nil {
		e = ApiError{
			Error: ERR_INTERNAL_TRYAGAIN,
		}
		d = "{ \"errcode\": \"ERR_INTERNAL_TRYAGAIN\" }"

	} else {
		d = string(dat)
	}
	http.Error(w, d, apiErrorCodes[e.Error])
}
