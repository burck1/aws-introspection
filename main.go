package main

import (
	"flag"
	"io"
	"log"
	"net/http"
	"time"
)

func main() {
	start := time.Now().Format(time.Stamp)

	serverPtr := flag.Bool("server", false, "host as a http server")
	portPtr := flag.String("port", "80", "the port to host the server at")

	flag.Parse()

	log.Println("start:", start)
	log.Println("server:", *serverPtr)
	log.Println("port:", *portPtr)

	if *serverPtr {
		// setup as server
		http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			now := time.Now().Format(time.Stamp)
			_, err := io.WriteString(w, now+"\n")
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
		})

		err := http.ListenAndServe(":"+*portPtr, nil)
		log.Fatal(err)
	} else {
		// output to command line
	}
}
