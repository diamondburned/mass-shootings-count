package gva

import (
	"context"
	"encoding/json"
	"io"
	"net/http"
	"os"
	"testing"

	"github.com/go-test/deep"
)

func TestScraperMassShootings(t *testing.T) {
	s := NewScraper()
	s.Client.Transport = roundTripFunc(func(r *http.Request) (*http.Response, error) {
		if r.URL.String() != "https://www.gunviolencearchive.org/reports/mass-shooting?page=0" {
			return newResponse(r, 404, nil), nil
		}
		body := mustTestData(t, "testdata/mass-shooting.html")
		return newResponse(r, 200, body), nil
	})

	records, err := s.MassShootings(context.Background(), 0)
	if err != nil {
		t.Fatal(err)
	}

	var expect []MassShootingRecord
	if err := json.NewDecoder(mustTestData(t, "testdata/mass-shooting.json")).Decode(&expect); err != nil {
		t.Fatal("cannot unmarshal json test data:", err)
	}

	if diff := deep.Equal(records, expect); diff != nil {
		t.Error(diff)
	}
}

func mustTestData(t *testing.T, path string) io.Reader {
	f, err := os.Open(path)
	if err != nil {
		t.Fatal("cannot open test data:", err)
	}
	t.Cleanup(func() { f.Close() })
	return f
}

type roundTripFunc func(r *http.Request) (*http.Response, error)

func (f roundTripFunc) RoundTrip(r *http.Request) (*http.Response, error) {
	return f(r)
}

func newResponse(r *http.Request, status int, body io.Reader) *http.Response {
	return &http.Response{
		Status:        http.StatusText(status),
		StatusCode:    status,
		Proto:         "HTTP/1.1",
		ProtoMajor:    1,
		ProtoMinor:    1,
		Header:        http.Header{},
		Body:          io.NopCloser(body),
		ContentLength: -1,
		Request:       r,
	}
}
