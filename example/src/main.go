package main

import (
	"net/http"
	"log"
	"encoding/json"
	"fmt"
)


type Request struct {
	Type int `json:"type"`
}

func handle(rw http.ResponseWriter, req *http.Request) {
	decoder := json.NewDecoder(req.Body)
	var data Request
	err := decoder.Decode(&data); if err != nil {
		log.Panic(err)
	}
	if data.Type == 0 {
		rw.WriteHeader(200)
		fmt.Fprint(rw, "Hello!")
	} else if data.Type == 1 {
		rw.WriteHeader(204)
	} else {
		rw.WriteHeader(404)
	}
}

func main() {
	http.HandleFunc("/", handle)
	log.Fatal(http.ListenAndServe(":8082", nil))
}
