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

func (c *Controller) register_user() {
	c.register_path("/v1/user", c.DefaultInvalidMethodHandler)
	c.register_path("POST /v1/user", c.user_post)
	c.register_path("PATCH /v1/user", c.user_patch)

	c.register_path("/v1/user/fetch", c.DefaultInvalidMethodHandler)
	c.register_path("POST /v1/user/fetch", c.user_fetch_post)

	c.register_path("/v1/user/auth", c.DefaultInvalidMethodHandler)
	c.register_path("POST /v1/user/auth", c.user_auth_post)
	c.Whitelist["/v1/user/auth"] = true
	c.register_path("DELETE /v1/user/auth", c.user_auth_delete)
	c.register_path("GET /v1/user/auth", c.user_auth_get)
}

func (s *Controller) user_auth_post(w http.ResponseWriter, r *http.Request) {
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
func (s *Controller) user_auth_delete(w http.ResponseWriter, r *http.Request) {
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

func (s *Controller) user_auth_get(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func (s *Controller) user_fetch_post(w http.ResponseWriter, r *http.Request) {
	type BodyType struct {
		Users *[]int `json:"users,omitempty"`
		Limit *int   `default:"5" json:"limit,omitempty"`
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
	//TODO FETCH AUTHKEY
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

	var acc_arr []int

	if req.Users == nil || len(*req.Users) == 0 {
		acc_arr = []int{idreq}
		log.Println(idreq)
	} else {
		acc_arr = make([]int, len(*req.Users))
		for key, val := range *req.Users {
			acc_arr[key] = val
		}
	}
	db := database.GetDBConn()
	query := `
SELECT * FROM public.UserFetch(
  p_userIds     := $1::INTEGER[], 
  p_requesterId := $2::INTEGER); 
  
  `
	rows, err := db.DB.Query(context.Background(), query, acc_arr, idreq)

	if err != nil {
		log.Println(err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		db.Release()
		return
	}
	defer rows.Close()

	type BaseReturn struct {
		Found         bool    `json:"found"`
		IsBanned      *bool   `json:"isBanned,omitempty"`
		IsAdmin       *bool   `json:"isAdmin,omitempty"`
		AdminId       *int    `json:"adminId,omitempty"`
		Name          *string `json:"name,omitempty"`
		ProfilPicture *string `json:"profilPicture,omitempty"`
		Email         *string `json:"email,omitempty"`
		City          *string `json:"city,omitempty"`
		Municipality  *string `json:"municipality,omitempty"`
		PostalCode    *string `json:"postalCode,omitempty"`
	}

	type MajorBaseReturn struct {
		Users map[int]BaseReturn `json:"users"`
	}
	re := MajorBaseReturn{
		Users: make(map[int]BaseReturn),
	}
	for rows.Next() {
		var requestedUserId int
		var found bool
		var isBanned *bool
		var isAdmin *bool
		var adminId *int
		var name *string
		var profilPicture *string
		var email *string
		var city *string
		var municipality *string
		var postalCode *string

		err := rows.Scan(&requestedUserId, &found, &isBanned, &isAdmin, &adminId, &name, &profilPicture, &email, &city, &municipality, &postalCode)
		if err != nil {
			log.Println(err)
			commons.ApiError{
				Error:     commons.ERR_INTERNAL_DB_FAIL,
				Errorinfo: err,
			}.HTTPSend(w)
			db.Release()
			return
		}

		re.Users[requestedUserId] = BaseReturn{
			Found:         found,
			IsBanned:      isBanned,
			IsAdmin:       isAdmin,
			AdminId:       adminId,
			Name:          name,
			ProfilPicture: profilPicture,
			Email:         email,
			City:          city,
			Municipality:  municipality,
			PostalCode:    postalCode,
		}
	}

	db.Release()
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
}

func (s *Controller) user_post(w http.ResponseWriter, r *http.Request) {
	type BodyType struct {
		Lastname  *string `json:"lastname,omitempty"`
		Firstname *string `json:"firstname,omitempty"`
		Email     *string `json:"email,omitempty"`
		Password  *string `json:"password,omitempty"`
		Poste     string  `default:" " json:"post,omitempty"`
		Role      string  `default:"employe" json:"role,omitempty"`
	}
	var authkey uuid.UUID
	{
		a, err := s.fetch_auth(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}

		authkey = a
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
	{
		re := []string{}
		if req.Lastname == nil {
			re = append(re, "lastname")
		}
		if req.Firstname == nil {
			re = append(re, "firstname")
		}
		if req.Email == nil {
			re = append(re, "email")
		} else {
			//TODO CHECK FOR INVALIDITY .
		}
		switch req.Role {
		case "":
			req.Role = "employe"
			break
		case "employe", "admin", "directeur":
			break
		default:
			re = append(re, "role")
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

	// Handle Req

	db := database.GetDBConn()

	query := `
  SELECT * 
  FROM userCreate( 
    $1::BOOL,
	$2::UUID,
	$3::TEXT,
	$4::TEXT,
    $5::TEXT,
	$6::TEXT,
	$7::TEXT,
	$8::typerole
  );`

	rows, err := db.DB.Query(context.Background(), query,
		commons.DebugMode,
		authkey,
		req.Lastname,
		req.Firstname,
		req.Poste,
		req.Email,
		req.Password,
		req.Role)

	if err != nil {
		log.Println(err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		db.Release()
		return
	}
	defer rows.Close()

	type MajorBaseReturn struct {
		Id string `json:"id"`
	}
	re := MajorBaseReturn{}
	var allowed bool = false
	if rows.Next() {
		var id uuid.UUID

		err := rows.Scan(&allowed, &id)
		if err != nil {
			log.Println(err)
			commons.ApiError{
				Error:     commons.ERR_INTERNAL_DB_FAIL,
				Errorinfo: err,
			}.HTTPSend(w)
			db.Release()
			return
		}
		re.Id = id.String()
	} else {
		log.Println(err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: rows.Err(),
		}.HTTPSend(w)
		db.Release()
		return
	}

	db.Release()

	if !allowed {
		commons.ApiError{
			Error: commons.ERR_AUTH_NO_PERMISSION,
		}.HTTPSend(w)
		return
	}
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
}
func (s *Controller) user_patch(w http.ResponseWriter, r *http.Request) {
	type BodyType struct {
		Idutil    *string `json:"idutil,omitempty"`
		Lastname  *string `json:"lastname,omitempty"`
		Firstname *string `json:"firstname,omitempty"`
		Email     *string `json:"email,omitempty"`
		Password  *string `json:"password,omitempty"`
		Poste     *string `json:"post,omitempty"`
		Role      *string `json:"role,omitempty"`
	}
	var authkey uuid.UUID
	var b []byte

	{
		a, err := s.fetch_auth(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}
		authkey = a
	}
	{
		a, err := s.fetch_body(r)
		if err != nil {
			err.HTTPSend(w)
			return
		}
		b = a
	}
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
	idutil := uuid.UUID{}
	{
		re := []string{}
		if req.Idutil == nil {
			re = append(re, "Idutil")
		} else if idutil.Scan(*req.Idutil) != nil {
			commons.ApiError{
				Error:     commons.ERR_REQ_BODY_INVALID,
				Errorinfo: `{ "received_body": "` + string(b) + `" }`,
			}.HTTPSend(w)
			return
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

	// Handle Req

	db := database.GetDBConn()

	query := `
SELECT * 
FROM userUpdate( 
	$1::BOOL,
	$2::UUID,
	$3::UUID,
	$4::TEXT,
	$5::TEXT,
	$6::TEXT,
	$7::TEXT,
	$8::TEXT,
	$9::typerole
);
`
	rows, err := db.DB.Query(context.Background(), query,
		commons.DebugMode,
		authkey,
		idutil,
		req.Lastname,
		req.Firstname,
		req.Poste,
		req.Email,
		req.Password,
		req.Role)

	if err != nil {
		log.Println(err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)
		db.Release()
		return
	}
	defer rows.Close()

	type MajorBaseReturn struct {
		Found bool `json:"found"`
	}
	re := MajorBaseReturn{}

	var allowed bool = false
	var found bool = false

	if rows.Next() {

		err := rows.Scan(&allowed, &found)
		if err != nil {
			log.Println(err)
			commons.ApiError{
				Error:     commons.ERR_INTERNAL_DB_FAIL,
				Errorinfo: err,
			}.HTTPSend(w)
			db.Release()
			return
		}
		re.Found = found
	} else {
		log.Println(rows.Err())
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: rows.Err(),
		}.HTTPSend(w)
		db.Release()
		return
	}

	db.Release()

	if !allowed {
		commons.ApiError{
			Error: commons.ERR_AUTH_NO_PERMISSION,
		}.HTTPSend(w)
		return
	}
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
}
