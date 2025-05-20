package server

import (
	"encoding/json"
	"evedem_api/internal/commons"
	"net/http"
)

func (c *Controller) register_db() {
	c.register_path("GET /v1/db/health", c.db_health)
}

func (s *Controller) db_health(w http.ResponseWriter, r *http.Request) {
	ret, err := json.Marshal("")
	if err != nil {
		commons.ApiError{
			Error:     commons.ERR_INTERNAL_TRYAGAIN,
			Errorinfo: "Unable to Generate Body",
			Data:      nil,
		}.HTTPSend(w)
		return
	}
	if _, err := w.Write(ret); err != nil {
		return
	}
}
