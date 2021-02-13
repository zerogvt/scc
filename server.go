package main

import (
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strconv"
	"strings"
)

// App represents the server's internal state.
// It holds configuration about providers and content
type App struct {
	ContentClients map[Provider]Client
	Config         ContentMix
}

func (a App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("%s %s", req.Method, req.URL.String())
	m, err := url.ParseQuery(req.URL.RawQuery)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	count, err := strconv.Atoi(m["count"][0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	offset, err := strconv.Atoi(m["offset"][0])
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	var i, pi int
	if offset != 0 {
		if offset <= len(a.Config) {
			pi = len(a.Config) - offset
		} else {
			pi = offset % len(a.Config)
		}
	}
	fmt.Println(pi)
	news := []*ContentItem{}
	for i = 0; i < count; i++ {
		items := []*ContentItem{}
		prov := a.Config[pi]
		fmt.Println(prov)
		if items, err = a.ContentClients[prov.Type].GetContent("todo_ip", 1); err != nil {
			if items, err = a.ContentClients[*prov.Fallback].GetContent("todo_ip", 1); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				//TODO
				w.Write([]byte("todo_write_so_far"))
				return
			}
		}
		news = append(news, items...)
		pi = (pi + 1) % len(a.Config)
	}
	builder := strings.Builder{}
	for _, n := range news {
		if _, err = builder.WriteString(n.Source + " "); err != nil {
			w.Write([]byte(builder.String()))
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	w.WriteHeader(http.StatusOK)
	w.Write([]byte(builder.String()))
}
