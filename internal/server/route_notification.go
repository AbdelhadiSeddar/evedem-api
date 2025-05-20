package server

import (
	"context"
	"encoding/json"
	"evedem_api/internal/commons"
	"evedem_api/internal/database"
	"log"
	"net/http"

	"github.com/google/uuid"
)

func (c *Controller) register_notification() {
	c.register_path("/v1/noti", c.DefaultInvalidMethodHandler)
	c.register_path("POST /v1/noti", c.noti_post)

	c.register_path("/v1/noti/fetch", c.DefaultInvalidMethodHandler)
	c.register_path("POST /v1/noti/fetch", c.noti_fetch_post)
}

func (s *Controller) noti_post(w http.ResponseWriter, r *http.Request) {
	type BodyType struct {
		Email    *string `json:"email,omitempty"`
		Password *string `json:"password,omitempty"`
	}
	b, err := s.fetch_body(r)
	if err != nil {
		err.HTTPSend(w)
		return
	}
	req := BodyType{}

	if err := json.Unmarshal(b, &req); err != nil {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_INVALID,
			Errorinfo: `{ "received_body": "` + string(b) + `" }`,
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	{
		re := []string{}
		if req.Email == nil {
			re = append(re, "email")
		} else {
		}

		if req.Password == nil {
			re = append(re, "password")
		}
		if len(re) > 0 {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_MISSING,
				Errorinfo: re,
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}

	idses, err := s.SessionRegister(*req.Email, *req.Password)

	if err != nil && err.Error != commons.ERR_AUTH_DEAD {
		err.HTTPSend(w)
		return
	}

	// Generate a response
	re := map[string]string{}
	if idses != nil {
		re["login"] = "success"
		re["authkey"] = idses.String()

		ret, err := json.Marshal(re)
		if err != nil {
			commons.ApiError{
				Error:     commons.ERR_INTERNAL_TRYAGAIN,
				Errorinfo: "Unable to generate body.",
			}.HTTPSend(w)
			return
		}

		if _, err := w.Write(ret); err != nil {
			return
		}
	} else {
		if err != nil {
			if err.Error == commons.ERR_AUTH_DEAD {
				re["login"] = "banned"
			} else {
				re["login"] = "invalid"
			}
		} else {
			re["login"] = "invalid"
		}

		ret, err := json.Marshal(re)
		if err != nil {
			commons.ApiError{
				Error:     commons.ERR_INTERNAL_TRYAGAIN,
				Errorinfo: "Unable to generate body.",
			}.HTTPSend(w)
			return
		}

		http.Error(w, string(ret), 401)
		return
	}

}
func (s *Controller) noti_delete(w http.ResponseWriter, r *http.Request) {
	auth, err := s.fetch_auth(r)
	if err != nil {
		err.HTTPSend(w)
		return
	}
	{
		err := s.SessionRevoke(auth)
		if err != nil {
			err.HTTPSend(w)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}



func (s *Controller) noti_fetch_post(w http.ResponseWriter, r *http.Request) {
	type BodyType struct {
		Before *string `json:"before,omitempty"` // RFC3339 string
		After  *string `json:"after,omitempty"`  // RFC3339 string
		Limit  *int   `json:"limit,omitempty"`  //  If nil, the DB default takes over
	}
	var b []byte
	{
		var err *commons.ApiError
		b, err = s.fetch_body(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}
	}

	var idreq int = -1
	var authkey uuid.UUID
	{
		a, _ := s.fetch_auth(r)
		authkey = a
	}
	idreq = *s.Ac.GetUserUUID(authkey)

	req := BodyType{}
	if len(b) > 0 {
		if err := json.Unmarshal(b, &req); err != nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: `{ "received_body": "` + string(b) + `" }`,
				Data:      nil,
			}.HTTPSend(w)
			return
		}
	}

	db := database.GetDBConn()

	// Construct query parameters based on input. Handle nil values correctly
	var beforeParam any = nil // Null for DB
	if req.Before != nil {
		beforeParam = *req.Before
	}

	var afterParam any = nil // Null for DB
	if req.After != nil {
		afterParam = *req.After
	}
    var limitParam any = nil
    if req.Limit !=nil{
      limitParam = *req.Limit
    }

	query := `
SELECT COALESCE(json_agg(row_to_json(t)), '{}'::json)
FROM (
    SELECT "notificationId", content, date
    FROM public.NotificationFetch(
        p_notified_id := $1,
        p_before := $2,
        p_after := $3,
        p_limit := $4
    )
) t;
`
	var notificationJson []byte
	err := db.DB.QueryRow(context.Background(), query, idreq, beforeParam, afterParam, limitParam).Scan(notificationJson)

	if err != nil {
		log.Println(err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		db.Release()
		return
	}
	db.Release()

	if _, err := w.Write(notificationJson); err != nil {
		return
	}
}
