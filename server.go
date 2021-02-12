package main

import (
	"fmt"
	"log"
	"net/http"
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
	var i, count, offset, provnum int64
	var err error
	params := req.URL.Query()
	fmt.Println(params)
	if count, err = strconv.ParseInt(params.Get("count"), 0, 64); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if offset, err = strconv.ParseInt(params.Get("offset"), 0, 64); err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	if offset != 0 {
		provnum = int64(len(a.Config)) % offset
	}
	news := []*ContentItem{}
	for i = 0; i < count; i++ {
		items := []*ContentItem{}
		prov := a.Config[provnum]
		if items, err = a.ContentClients[prov.Type].GetContent("todo_ip", 1); err != nil {
			if items, err = a.ContentClients[*prov.Fallback].GetContent("todo_ip", 1); err != nil {
				w.WriteHeader(http.StatusInternalServerError)
				//TODO
				w.Write([]byte("todo_write_so_far"))
				return
			}
		}
		news = append(news, items...)
		provnum = int64(len(a.Config)) % (provnum + 1)
	}
	builder := strings.Builder{}
	for _, n := range news {
		if _, err = builder.WriteString(n.Source + " "); err != nil {
			w.Write([]byte(builder.String()))
			w.WriteHeader(http.StatusInternalServerError)
		}
	}
	w.Write([]byte(builder.String()))
	w.WriteHeader(http.StatusOK)
}
