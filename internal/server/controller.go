package server

import (
	"io"
	"log"
	"net/http"
	"os"

	"evedem_api/internal/commons"

	"github.com/google/uuid"
)

type Controller struct {
	S   *Server
	Mux *http.ServeMux
	// Map where String is the path, and controller path has all whats required for a callback
	Ac        *AuthCache
	Whitelist map[string]bool
}

var DefaultController = Controller{
	Mux:       http.NewServeMux(),
	Ac:        NewAuthCache(1800),
	Whitelist: make(map[string]bool, 0),
}

func (c *Controller) register_routes() {
	c.register_user()
	c.register_db()
	c.register_notification()
	c.register_categories()
  c.register_products()
}

type handler func(http.ResponseWriter, *http.Request)

func (c *Controller) register_path(path string, f handler) {
	c.Mux.HandleFunc(path, f)
}
func (c *Controller) DefaultInvalidMethodHandler(w http.ResponseWriter, r *http.Request) {
	commons.ApiError{
		Error:     commons.ERR_REQ_METHOD_INVALID,
		Errorinfo: r.Method + " Invalid.",
	}.HTTPSend(w)
	return
}
func (c *Controller) fetch_auth(r *http.Request) (uuid.UUID, *commons.ApiError) {
	if commons.DebugMode {
		return uuid.Nil, nil
	}
	a := r.Header.Get("Authorization")
	if a == "" {
		return uuid.Nil, &commons.ApiError{
			Error:     commons.ERR_AUTH_REQUIRED,
			Errorinfo: "Please Login",
		}
	}
	var auth uuid.UUID

	if err := auth.Scan(a); err != nil {
		return uuid.Nil, &commons.ApiError{
			Error:     commons.ERR_AUTH_INVALID,
			Errorinfo: "Please Login Properly",
		}
	}
	return auth, nil
}
func (c *Controller) fetch_body(r *http.Request) ([]byte, *commons.ApiError) {
	b := []byte{}
	if r.Method != "GET" && r.Method != "Head" {
		i, err := io.ReadAll(r.Body)
		b = i
		if err != nil || len(b) <= 0 {
			return nil, &commons.ApiError{
				Error:     commons.ERR_REQ_BODY_EMPTY,
				Errorinfo: nil,
				Data:      nil,
			}
		}
	}
	return b, nil
}

func (c *Controller) ControllerMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Set CORS headers
		w.Header().Set("Access-Control-Allow-Origin", os.Getenv("EVERDEEM_FRONT_URL"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS, PATCH")
		w.Header().Set("Access-Control-Allow-Headers", "Accept, Authorization, Content-Type, Content-Length")
		w.Header().Set("Access-Control-Allow-Credentials", "false")
		// Handle preflight OPTIONS requests
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		log.Println(r.Header.Get("X-Forwarded-For"), " Received ", r.Method, r.URL)
		if commons.DebugMode &&
			(r.Header.Get("X-Forwarded-For") != "" || r.Header.Get("X-Real-IP") != "") {
			w.WriteHeader(http.StatusServiceUnavailable)
			return
		}

		authkey := uuid.Nil

		if !commons.DebugMode {
			log.Println(r.URL.Path)
			if val, ok := c.Whitelist[r.URL.Path]; !ok || !val {
				aut, err := c.fetch_auth(r)
				if err != nil {
					err.HTTPSend(w)
					return
				}

				authkey = aut
				apierr := c.AuthVerif(authkey)
				if apierr != nil {
					apierr.HTTPSend(w)
					return
				}

			}
		}
		// Proceed with the next handler
		next.ServeHTTP(w, r)
	})
}
