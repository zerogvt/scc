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
	count, offset, err := getParams(req)
	if err != nil {
		w.WriteHeader(http.StatusBadRequest)
		return
	}
	ip := getIP(req)
	// Make a channel for each content Provider  mirroring our contentmix config.
	// That way we can download content concurrently from all of them.
	chans := make([]chan []*ContentItem, len(a.Config))
	for i, cfg := range a.Config {
		// Use unbuffered channels.
		// We could utilize buffered ones if we want more concurrency but
		// finding the sweet spot between max throughput and
		// clogging the providers might be tricky.
		c := make(chan []*ContentItem)
		chans[i] = c
		// for each channel (i.e. content provider in out contentmix)
		// start a goroutine to constantly pull off it
		go func(c chan []*ContentItem, cfg ContentConfig) {
			for {
				items := []*ContentItem{}
				// try main provider in this config
				if items, err = a.ContentClients[cfg.Type].GetContent(ip, 1); err != nil {
					// If main provider failed and there's no Fallback one then close our channel.
					if a.ContentClients[*cfg.Fallback] == nil {
						close(c)
						return
					}
					// If a Fallback exist try to get item off it and if that fails too then close our channel.
					if items, err = a.ContentClients[*cfg.Fallback].GetContent(ip, 1); err != nil {
						close(c)
						return
					}
				}
				c <- items
			}
		}(c, cfg)
	}
	// Loop through providers in our config (contentmix) reading items off their channels.
	// pi is provider index.
	pi := offset % len(a.Config)
	news := []*ContentItem{}
	for i := 0; i < count; i++ {
		items, ok := <-chans[pi]
		if !ok {
			// Current channel/content provider bailed.
			// As per specs return as much as we have so far and return.
			writeRes(news, w)
			return
		}
		news = append(news, items...)
		pi = (pi + 1) % len(a.Config)
	}
	writeRes(news, w)
}

func getParams(r *http.Request) (int, int, error) {
	var count, offset int
	var err error
	var v url.Values
	v, err = url.ParseQuery(r.URL.RawQuery)
	if err != nil {
		return count, offset, err
	}
	if _, ok := v["count"]; ok {
		count, err = strconv.Atoi(v["count"][0])
		if err != nil {
			return count, offset, err
		}
	}
	if _, ok := v["offset"]; ok {
		offset, err = strconv.Atoi(v["offset"][0])
	}
	return count, offset, err
}

func getIP(r *http.Request) string {
	// Try to get the real ip.
	if ip := r.Header.Get("X-FORWARDED-FOR"); ip != "" {
		return ip
	}
	// Fall back to the remote addr if no real ip can be retrieved.
	return r.RemoteAddr
}

func writeRes(news []*ContentItem, w http.ResponseWriter) {
	body, err := json.Marshal(news)
	if err != nil {
		w.WriteHeader(http.StatusInternalServerError)
		return
	}
	if _, err := w.Write(body); err != nil {
		w.WriteHeader(http.StatusInternalServerError)
	}
}
