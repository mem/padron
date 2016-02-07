package main

import (
	"fmt"
	"log"
	"net/http"
	"server"
)

func main() {
	server.RegisterHandlers()
	http.Handle("/", http.FileServer(http.Dir("static")))
	log.Print("Trying port 80")
	err := http.ListenAndServe("0.0.0.0:80", nil)
	for port := 8080; err != nil && port < 8090; port++ {
		log.Printf("Trying port %d", port)
		addr := fmt.Sprintf("0.0.0.0:%d", port)
		err = http.ListenAndServe(addr, nil)
	}
	log.Fatal(err)
}
