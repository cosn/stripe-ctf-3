package main

import (
	"fmt"
	"log"
	"net/http"
	"os"
	"runtime/pprof"
	"strings"
)

type Server interface {
	Init(id, port int)
	Health(rw http.ResponseWriter, req *http.Request)
	Indexed(rw http.ResponseWriter, req *http.Request)
	Index(rw http.ResponseWriter, req *http.Request)
	Query(rw http.ResponseWriter, req *http.Request)
}

type SearchServer struct {
	id, port     int
	indexed      bool
	index        *Index
	cprof, mprof string
}

type MasterServer struct {
	c []*SearchClient
}

const clients = 3
const idxTimeout = 125

func (s *MasterServer) Init(id, port int) {
	s.c = make([]*SearchClient, clients)

	for i := 0; i < clients; i++ {
		c := new(SearchClient)
		c.Init(basePort+i+1, i+1)
		s.c[i] = c
	}
}

func (s *MasterServer) Health(rw http.ResponseWriter, req *http.Request) {
	succ := true

	for _, client := range s.c {
		if !client.Health() {
			log.Printf("Server %d is not up\n", client.id)
			succ = false
		}
	}

	if succ {
		rw.Write(success())
		rw.WriteHeader(http.StatusOK)
	} else {
		rw.Write(fail("All nodes are not up"))
		rw.WriteHeader(http.StatusBadGateway)
	}
}

func (s *MasterServer) Indexed(rw http.ResponseWriter, req *http.Request) {
	succ := true

	for _, client := range s.c {
		if !client.Indexed() {
			log.Printf("Server %d is not indexed\n", client.id)
			succ = false
		}
	}

	if succ {
		rw.Write(success())
		rw.WriteHeader(http.StatusOK)
	} else {
		rw.Write(fail("Nodes are not indexed"))
		rw.WriteHeader(http.StatusBadGateway)
	}
}

func (s *MasterServer) Index(rw http.ResponseWriter, req *http.Request) {
	path := req.FormValue("path")
	if len(path) == 0 {
		rw.Write(fail("path is missing"))
		rw.WriteHeader(http.StatusBadRequest)
	}

	for _, client := range s.c {
		go client.Index(path)
	}

	rw.Write(success())
	rw.WriteHeader(http.StatusOK)
}

func (s *MasterServer) Query(rw http.ResponseWriter, req *http.Request) {
	q := req.FormValue("q")
	if len(q) == 0 {
		rw.Write(fail("query is missing"))
		rw.WriteHeader(http.StatusBadRequest)
	}

	rw.Write(s.client(int(q[0])).Query(q))
	rw.WriteHeader(http.StatusOK)
}

func (s *MasterServer) client(c int) *SearchClient {
	return s.c[c%clients]
}

func (s *SearchServer) Init(id, port int) {
	s.id = id
	s.port = port
	s.index = new(Index)
	s.index.Init(id)
}

func (s *SearchServer) Health(rw http.ResponseWriter, req *http.Request) {
	rw.Write(success())
	rw.WriteHeader(http.StatusOK)
}

func (s *SearchServer) Indexed(rw http.ResponseWriter, req *http.Request) {
	if s.indexed {
		rw.Write(success())
	} else {
		rw.Write(fail("Not indexed"))
	}

	rw.WriteHeader(http.StatusOK)
}

func (s *SearchServer) Index(rw http.ResponseWriter, req *http.Request) {
	s.indexed = false
	if len(s.cprof) > 0 {
		f, err := os.Create(fmt.Sprintf("%d-%v", s.id, s.cprof))
		if err != nil {
			log.Fatal(err)
		}
		pprof.StartCPUProfile(f)
		defer pprof.StopCPUProfile()
	}

	s.index.Index(req.FormValue("path"))

	if len(s.mprof) > 0 {
		f, err := os.Create(fmt.Sprintf("%d-%v", s.id, s.mprof))
		if err != nil {
			log.Fatal(err)
		}

		pprof.WriteHeapProfile(f)
		f.Close()
	}
	s.indexed = true
	rw.Write(success())
	rw.WriteHeader(http.StatusOK)
}

func (s *SearchServer) Query(rw http.ResponseWriter, req *http.Request) {
	q := req.FormValue("q")
	log.Printf("%d: searching for %q\n", s.id, q)
	if res := s.index.Search(q); res != nil {
		rw.Write(successQuery(res))
	}

	rw.WriteHeader(http.StatusOK)
}

func success() []byte {
	return []byte("{\"success\": true}")
}

func successQuery(res []string) []byte {
	return []byte("{\"success\": true,\n\"results\": [" + strings.Join(res, ",\n") + "]}")
}

func fail(msg string) []byte {
	return []byte("{\"success\": false, \"error\": \"" + msg + "\"}")
}
