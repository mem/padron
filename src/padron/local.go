package main

import (
	"net/http"
	"padron/server"
	"log"
)

func main() {
	server.RegisterHandlers()
	http.Handle("/", http.FileServer(http.Dir("static")))
	log.Print("Trying port 80")
	err := http.ListenAndServe("0.0.0.0:80", nil)
	if err != nil {
		log.Print("Trying port 8080")
		log.Fatal(http.ListenAndServe("0.0.0.0:8080", nil))
	}
}
