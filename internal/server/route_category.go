package server

import (
	"context"
	"encoding/json"
	"evedem_api/internal/commons"
	"evedem_api/internal/database"
	"log"
	"net/http"

)

func (c *Controller) register_categories() {
	c.register_path("/v1/categories", c.DefaultInvalidMethodHandler)
	c.register_path("POST /v1/categories/products", c.categories_products_post)
  c.Whitelist["/v1/categories/products"] = true
}

func (s *Controller) categories_products_post(w http.ResponseWriter, r *http.Request) {
	type BodyType struct {
		CatId     *int    `json:"cat_id,omitempty"`
		After     *string `json:"after,omitempty"`  // RFC3339 string
		Before    *string `json:"before,omitempty"` // RFC3339 string
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

	if req.CatId == nil {
		commons.ApiError{
			Error:     commons.ERR_REQ_BODY_MISSING,
			Errorinfo: "cat_id is required",
			Data:      nil,
		}.HTTPSend(w)
		return
	}

	db := database.GetDBConn()
  defer db.Release()

	var afterParam any = nil
	if req.After != nil {
		afterParam = *req.After
	}

	var beforeParam any = nil
	if req.Before != nil {
		beforeParam = *req.Before
	}

	query := `SELECT GetCategoryProducts($1, $2, $3);`

	var productsJson []byte
	err := db.DB.QueryRow(context.Background(), query, *req.CatId, afterParam, beforeParam).Scan(&productsJson)
	if err != nil {
		log.Println(err)
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_DB_FAIL,
			Errorinfo: err,
		}.HTTPSend(w)

		return
	}

	if _, err := w.Write(productsJson); err != nil {
		return
	}
}
