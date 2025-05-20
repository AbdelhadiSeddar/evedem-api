package server

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	_ "github.com/joho/godotenv/autoload"

	"evedem_api/internal/database"
)

type Server struct {
	port int
}

func NewServer() *http.Server {
	port, _ := strconv.Atoi(os.Getenv("EVERDEEM_PORT"))
	DefaultController.S = &Server{
		port: port,
	}
	database.ConfigDatabaseConnections()
	db := database.GetDBConn()

	empty, missing, err := db.CheckTables()

	if !empty {
		if len(missing) > 0 {
			panic("Missing Tables : " + strings.Join(missing, ", "))
		} else if err != nil {
			panic("Internal Error : " + string(err.Error) + " ")
		}
	} else {
		log.Println("Database Is Empty")
		err := db.CreateTables()
		if err != nil {
			panic("Internal Error while creating: " + string(err.Error))
		}
	}
	log.Println("Database is valid.")
	db.CreateFunctions()
	log.Println("Functions Created")

	db.Release()
	// Declare Server config
	server := &http.Server{
		Addr:         fmt.Sprintf(":%d", port),
		Handler:      DefaultController.RegisterRoutes(),
		IdleTimeout:  time.Minute,
		ReadTimeout:  10 * time.Second,
		WriteTimeout: 30 * time.Second,
	}

	return server
}
