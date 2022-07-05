package index

import (
	"context"
	"errors"
	"net/http"
	"time"

	"github.com/diamondburned/mass-shootings-count/cmd/days-since/frontend"
	"github.com/diamondburned/mass-shootings-count/gva"
	"github.com/diamondburned/mass-shootings-count/internal/watcher"
	"github.com/diamondburned/tmplutil"
	"github.com/go-chi/chi/v5"
	"github.com/go-chi/chi/v5/middleware"
)

var index = frontend.Template.Register("index", "index/index.html")

type handler struct {
	renderedData *watcher.Watcher[renderData]
}

const watchFlags = watcher.WatchAllowStale

const Day = 24 * time.Hour

func Mount(scraper *gva.Scraper) http.Handler {
	renderedData := watcher.Watch(5*time.Minute, watchFlags, func() (renderData, error) {
		records, err := scraper.MassShootings(context.Background(), 0)
		if err != nil {
			return renderData{}, err
		}

		if len(records) == 0 {
			return renderData{}, errors.New("no records found")
		}

		get := func(i int) ([]gva.MassShootingRecord, error) {
			if i == 0 {
				return records, nil
			}
			return scraper.MassShootings(context.Background(), i)
		}

		recordsToday, err := gva.MassShootingsOnDate(get, gva.Today())
		if err != nil {
			return renderData{}, err
		}

		now := time.Now()
		latestIncident := records[0].IncidentDate.AsTime(now)

		return renderData{
			Days:    int(now.Sub(latestIncident) / Day),
			Records: recordsToday,
		}, nil
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

	return r
}

type renderData struct {
	Days        int
	Records     []gva.MassShootingRecord
	LastUpdated time.Time
}

func (h handler) index(w http.ResponseWriter, r *http.Request) {
	renderData, err, t := h.renderedData.Get()
	if err != nil {
		w.WriteHeader(500)
		w.Write([]byte(err.Error()))
		return
	}

	renderData.LastUpdated = t

	index.Execute(w, renderData)
}
