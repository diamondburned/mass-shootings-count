package index

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"strconv"
	"time"

	"github.com/diamondburned/mass-shootings-count/cmd/mass-shootings-count/frontend"
	"github.com/diamondburned/mass-shootings-count/gva"
	"github.com/diamondburned/mass-shootings-count/internal/watcher"
	"github.com/diamondburned/tmplutil"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
	"github.com/timewasted/go-accept-headers"
)

var index = frontend.Template.Register("index", "index/index.html")

type handler struct {
	renderedData *watcher.Watcher[RenderData]
}

const watchFlags = watcher.WatchAllowStale

const Day = 24 * time.Hour

type cachedScraper struct {
	scraper *gva.Scraper
	records [][]gva.MassShootingRecord
}

func cacheScraper(scraper *gva.Scraper) *cachedScraper {
	return &cachedScraper{
		scraper: scraper,
		records: make([][]gva.MassShootingRecord, 0, 2),
	}
}

func (s *cachedScraper) MassShootings(ctx context.Context, i int) ([]gva.MassShootingRecord, error) {
	if len(s.records) < i {
		return s.records[i], nil
	}

	records, err := s.scraper.MassShootings(ctx, i)
	if err != nil {
		return nil, err
	}

	if i == len(s.records) {
		s.records = append(s.records, records)
	}

	return records, nil
}

func Mount(scraper *gva.Scraper) http.Handler {
	renderedData := watcher.Watch(2*time.Minute, watchFlags, func() (RenderData, error) {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Minute)
		defer cancel()

		scraper := cacheScraper(scraper)

		records, err := scraper.MassShootings(ctx, 0)
		if err != nil {
			return RenderData{}, err
		}

		if len(records) == 0 {
			return RenderData{}, errors.New("no records found")
		}

		today := gva.Today()
		yesterday := today.Add(-Day)

		var data RenderData
		data.LastUpdated = time.Now()

		data.Records.Today, err = gva.MassShootingsOnDate(ctx, scraper, today)
		if err != nil {
			return data, err
		}

		data.Records.Yesterday, err = gva.MassShootingsOnDate(ctx, scraper, yesterday)
		if err != nil {
			return data, err
		}

		now := time.Now()
		latestIncident := records[0].IncidentDate.AsTime(now)

		// Require a full day without any incidents in order for it to be
		// counted.
		data.DaysSince = int(now.Sub(latestIncident)/Day) - 1
		if data.DaysSince < 0 {
			data.DaysSince = 0
		}

		return data, nil
	})

	h := handler{
		renderedData: renderedData,
	}

	r := chi.NewRouter()
	r.Group(func(r chi.Router) {
		r.Use(tmplutil.AlwaysFlush)
		r.Use(middleware.NoCache)
		r.Get("/", h.index)
	})

	// Preload.
	h.renderedData.Get()

	return r
}

type RenderData struct {
	DaysSince   int
	LastUpdated time.Time
	Records     struct {
		Today     []gva.MassShootingRecord
		Yesterday []gva.MassShootingRecord
	}
}

func (h handler) index(w http.ResponseWriter, r *http.Request) {
	accepting, _ := accept.Negotiate(r.Header.Get("accept"),
		"text/html",
		"application/json",
	)

	switch accepting {
	case "application/json":
		w.Header().Set("Content-Type", "application/json")
		h.indexJSON(w, r)
	case "text/html":
		fallthrough
	default:
		w.Header().Set("Content-Type", "text/html")
		h.indexHTML(w, r)
	}
}

func (h handler) indexHTML(w http.ResponseWriter, r *http.Request) {
	type day struct {
		Name         string
		Time         time.Time
		Records      []gva.MassShootingRecord
		TotalInjured int
		TotalKilled  int
	}

	var data struct {
		RenderData
		Days    [2]day
		Refresh int // seconds
	}

	var err error

	data.RenderData, err = h.renderedData.Get()
	if err != nil {
		frontend.ErrorHTML(w, 503, err)
		return
	}

	q := r.URL.Query()
	if refresh, err := strconv.Atoi(q.Get("refresh")); err == nil {
		data.Refresh = refresh
	}

	data.Days = [2]day{
		{
			Name:    "Today",
			Time:    data.LastUpdated,
			Records: data.Records.Today,
		},
		{
			Name:    "Yesterday",
			Time:    data.LastUpdated.Add(-Day),
			Records: data.Records.Yesterday,
		},
	}

	for i, day := range data.Days {
		for _, rec := range day.Records {
			data.Days[i].TotalKilled += rec.NoKilled
			data.Days[i].TotalInjured += rec.NoInjured
		}
	}

	index.Execute(w, data)
}

func (h handler) indexJSON(w http.ResponseWriter, r *http.Request) {
	renderData, err := h.renderedData.Get()
	if err != nil {
		errorJSON(w, 503, err)
		return
	}

	if err := json.NewEncoder(w).Encode(renderData); err != nil {
		errorJSON(w, 500, err)
	}
}

func errorJSON(w http.ResponseWriter, code int, err error) {
	var resp struct {
		Error string
	}

	resp.Error = err.Error()

	w.WriteHeader(code)
	json.NewEncoder(w).Encode(resp)
}
