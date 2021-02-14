package main

import (
	"encoding/json"
	"log"
	"net/http"
	"net/url"
	"strconv"
)

// App represents the server's internal state.
// It holds configuration about providers and content
type App struct {
	ContentClients map[Provider]Client
	Config         ContentMix
}

func (a App) ServeHTTP(w http.ResponseWriter, req *http.Request) {
	log.Printf("%s %s", req.Method, req.URL.String())
	// this needs to precede any call to WriteHeader or Write
	// as per https://golang.org/pkg/net/http/#ResponseWriter
	w.Header().Set("Content-Type", "application/json")
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
	// make a channel for each content Provider
	chans := make([]chan []*ContentItem, len(a.Config))
	for i, cfg := range a.Config {
		c := make(chan []*ContentItem)
		chans[i] = c
		// and start goroutines to constantly pull from providers
		go func(c chan []*ContentItem, cfg ContentConfig) {
			for {
				items := []*ContentItem{}
				if items, err = a.ContentClients[cfg.Type].GetContent("todo_ip", 1); err != nil {
					if a.ContentClients[*cfg.Fallback] == nil {
						close(c)
						return
					}
					if items, err = a.ContentClients[*cfg.Fallback].GetContent("todo_ip", 1); err != nil {
						close(c)
						return
					}
				}
				c <- items
			}
		}(c, cfg)
	}
	// provider index
	pi := offset % len(a.Config)
	news := []*ContentItem{}
	for i := 0; i < count; i++ {
		items, ok := <-chans[pi]
		if !ok {
			w.WriteHeader(http.StatusInternalServerError)
			// todo write results so far
			return
		}
		news = append(news, items...)
		pi = (pi + 1) % len(a.Config)
	}
	body, err := json.Marshal(news)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
}
