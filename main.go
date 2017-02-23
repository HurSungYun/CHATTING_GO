package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"golang.org/x/net/http2"
)

type User struct {
	nick   string
	roomID string
}

type Message struct {
	roomID    string
	nick      string
	text      string
	timestamp time.Time
}

type Chatroom struct {
	ID       string
	title    string
	members  []*User
	numOfMsg int
}

func main() {
	cwd, err := os.Getwd()
	if err != nil {
		log.Fatal(err)
	}

	srv := &http.Server{
		Addr:    ":8000", // Normally ":443"
		Handler: http.FileServer(http.Dir(cwd)),
	}
	http2.ConfigureServer(srv, &http2.Server{})
	log.Fatal(srv.ListenAndServeTLS("server.crt", "server.key"))
}
