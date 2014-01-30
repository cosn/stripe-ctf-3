package main

import (
	"flag"
	"fmt"
	"log"
	"net/http"
)

const basePort = 9090

func main() {
	id := flag.Int("id", 0, "server number")
	master := flag.Bool("master", false, "master server")
	memprofile := flag.String("memprofile", "", "memory profiling")
	cpuprofile := flag.String("cpuprofile", "", "cpu profiling")

	flag.Parse()
	port := *id + basePort

	var s Server
	if *master {
		log.Printf("Starting master as server %d on port %d\n", *id, port)
		s = new(MasterServer)
	} else {
		log.Printf("Starting search as server %d on port %d\n", *id, port)
		s = &SearchServer{cprof: *cpuprofile, mprof: *memprofile}
	}

	s.Init(*id, port)

	http.HandleFunc("/healthcheck", s.Health)
	http.HandleFunc("/isIndexed", s.Indexed)
	http.HandleFunc("/index", s.Index)
	http.HandleFunc("/", s.Query)
	log.Fatal(http.ListenAndServe(fmt.Sprintf(":%d", port), nil))
}
