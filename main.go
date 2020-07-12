package main

import (
"flag"
"log"
"net/http"
)

var addr = flag.String("addr", ":8080", "http service address")
var cert = flag.String("cert", "server.crt", "tls certificate")
var key = flag.String("key", "server.key", "tls key")

func serveHome(w http.ResponseWriter, r *http.Request) {
	log.Println(r.URL)
	if r.URL.Path != "/" {
		http.Error(w, "Not found", http.StatusNotFound)
		return
	}
	if r.Method != "GET" {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return
	}
	http.ServeFile(w, r, "home.html")
}

func main() {
	flag.Parse()
	hub := newHub()
	go hub.run()
	http.HandleFunc("/", serveHome)
	http.HandleFunc("/ws", func(w http.ResponseWriter, r *http.Request) {
		serveWs(hub, w, r)
	})
	err := http.ListenAndServeTLS(*addr, *cert, *key, nil)
	if err != nil {
		log.Fatal("ListenAndServeTLS: ", err)
	}
}
