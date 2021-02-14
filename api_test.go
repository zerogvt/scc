package main

import (
	"encoding/json"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
)

type ErrProvider struct {
	Source Provider
}

func (ErrProvider) GetContent(userIP string, count int) ([]*ContentItem, error) {
	return []*ContentItem{}, errors.New("Provider Is Unreachable")
}

var (
	SimpleContentRequest = httptest.NewRequest("GET", "/?offset=0&count=5", nil)
	OffsetContentRequest = httptest.NewRequest("GET", "/?offset=5&count=5", nil)
)

func runRequest(t *testing.T, srv http.Handler, r *http.Request) (content []*ContentItem) {
	response := httptest.NewRecorder()
	srv.ServeHTTP(response, r)

	if response.Code != 200 {
		t.Fatalf("Response code is %d, want 200", response.Code)
		return
	}

	decoder := json.NewDecoder(response.Body)
	err := decoder.Decode(&content)
	if err != nil {
		t.Fatalf("couldn't decode Response json: %v", err)
	}

	return content
}

func TestResponseCount(t *testing.T) {
	content := runRequest(t, app, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

}

func TestResponseOrder(t *testing.T) {
	content := runRequest(t, app, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for i, item := range content {
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

func TestOffsetResponseOrder(t *testing.T) {
	content := runRequest(t, app, OffsetContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}

	for j, item := range content {
		i := j + 5
		if Provider(item.Source) != DefaultConfig[i%len(DefaultConfig)].Type {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

func TestProviderFallback(t *testing.T) {
	cfg := []ContentConfig{
		config1, config1, config2, config3, config1,
	}
	errapp := App{
		ContentClients: map[Provider]Client{
			Provider1: SampleContentProvider{Source: Provider1},
			Provider2: SampleContentProvider{Source: Provider2},
			Provider3: ErrProvider{Source: Provider3},
		},
		Config: cfg,
	}
	// Config: config1, config1, config2, config3, config4, config1 ...
	// Despite provider 3 in config 3 returning an err we should get a reply off
	// the fallback (prov 1) and go on normally
	content := runRequest(t, errapp, SimpleContentRequest)

	if len(content) != 5 {
		t.Fatalf("Got %d items back, want 5", len(content))
	}
	want := []Provider{Provider1, Provider1, Provider2, Provider1, Provider1}
	for i, item := range content {
		if Provider(item.Source) != want[i] {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, DefaultConfig[i].Type,
			)
		}
	}
}

func TestProviderError(t *testing.T) {
	cfg := []ContentConfig{
		config1, config1, config3, config2, config1,
	}
	errapp := App{
		ContentClients: map[Provider]Client{
			Provider1: SampleContentProvider{Source: Provider1},
			Provider2: ErrProvider{Source: Provider2},
			Provider3: ErrProvider{Source: Provider3},
		},
		Config: cfg,
	}
	// Now we should halt on config2 since both primary and fallback are set to fail
	// note: config3 should be ok due to fallback
	content := runRequest(t, errapp, SimpleContentRequest)

	if len(content) != 3 {
		t.Fatalf("Got %d items back, want 3", len(content))
	}
	want := []Provider{Provider1, Provider1, Provider1}
	for i, item := range content {
		if Provider(item.Source) != want[i] {
			t.Errorf(
				"Position %d: Got Provider %v instead of Provider %v",
				i, item.Source, cfg[i].Type,
			)
		}
	}
}
