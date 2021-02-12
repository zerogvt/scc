package main

import (
	"log"
	"net/http"
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
	items, _ := a.ContentClients[Provider1].GetContent("127.0.0.1", 2)
	items2, _ := a.ContentClients[Provider2].GetContent("127.0.0.1", 3)
	items = append(items, items2...)
	builder := strings.Builder{}
	for _, it := range items {
		builder.WriteString(it.Source + " ")
	}
	w.Write([]byte(builder.String()))
	w.WriteHeader(http.StatusNotImplemented)
}
