package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"net/http"
	"strings"
)

type SearchClient struct {
	url string
	id  int
	c   *http.Client
}

func (s *SearchClient) Init(port, id int) {
	s.url = fmt.Sprintf("http://localhost:%d", port)
	s.id = id
	s.c = new(http.Client)
}

func (s *SearchClient) Health() bool {
	defer func() {
		recover()
	}()

	r, err := s.c.Get(fmt.Sprintf("%v/healthcheck", s.url))
	if err != nil {
		log.Printf("client %d: health() error = %v\n", s.id, err)
		return false
	}

	if r.StatusCode != http.StatusOK {
		log.Printf("client %d: health() status code = %v\n", s.id, r.StatusCode)
		return false
	}

	return true
}

func (s *SearchClient) Indexed() bool {
	r, err := s.c.Get(fmt.Sprintf("%v/isIndexed", s.url))
	if err != nil {
		log.Printf("client %d: indexed() error = %v\n", s.id, err)
	}

	if r.StatusCode != http.StatusOK {
		log.Printf("client %d: indexed() status code = %v\n", s.id, r.StatusCode)
		return false
	}

	body, _ := ioutil.ReadAll(r.Body)
	if !strings.Contains(string(body), "true") {
		return false
	}

	return true
}

func (s *SearchClient) Index(path string) {
	r, err := s.c.Get(fmt.Sprintf("%v/index?path=%v", s.url, path))
	if err != nil {
		log.Printf("client %d: index() error = %v\n", s.id, err)
	}

	if r.StatusCode != http.StatusOK {
		log.Printf("client %d: index() status code = %v\n", s.id, r.StatusCode)
	}
}

func (s *SearchClient) Query(q string) []byte {
	r, err := s.c.Get(fmt.Sprintf("%v/?q=%v", s.url, q))
	if err != nil {
		log.Printf("client %d: query() error = %v\n", s.id, err)
	}

	if r.StatusCode != http.StatusOK {
		log.Printf("client %d: query() status code = %v\n", s.id, r.StatusCode)
	}

	body, _ := ioutil.ReadAll(r.Body)
	return body
}
