package server

import (
	"evedem_api/internal/commons"
	"net/http"
)

func (c *Controller) RegisterRoutes() http.Handler {
	DefaultController.register_routes()

	// Wrap the mux with CORS middleware
	return DefaultController.ControllerMiddleware(DefaultController.Mux)
}

func (s *Server) DefaultHandler(w http.ResponseWriter, r *http.Request) {
	commons.ApiError{
		Error:     commons.ERR_REQ_PATH_INVALID,
		Errorinfo: map[string]string{"received_path": r.RequestURI},
		Data:      nil,
	}.HTTPSend(w)
	return
}
