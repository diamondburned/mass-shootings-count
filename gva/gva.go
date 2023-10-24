package gva

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/pkg/errors"
)

// Date is a date type with no time.
type Date struct {
	_ [0]func() // incomparable
	t time.Time
}

// TZ is Eastern Time.
var TZ = func() *time.Location {
	l, err := time.LoadLocation("America/New_York")
	if err != nil {
		log.Panicln("cannot load location America/New_York:", err)
	}
	return l
}()

const datef = "January 2, 2006"

// Today returns the current Date.
func Today() Date {
	return NewDate(time.Now())
}

// NewDate creates a new Date from a time.
func NewDate(t time.Time) Date {
	return Date{t: t.In(TZ)}
}

// ParseDate parses a string of format "July 4, 2022".
func ParseDate(str string, loc *time.Location) (Date, error) {
	t, err := time.ParseInLocation(datef, str, loc)
	if err != nil {
		return Date{}, err
	}

	return Date{t: t}, nil
}

// AsTime converts the date into a Time using the clock within the time. The
// given clock time's timezone is used for the returned time.
func (d Date) AsTime(clock time.Time) time.Time {
	clock = clock.In(d.t.Location())

	return time.Date(
		d.t.Year(), d.t.Month(), d.t.Day(),
		clock.Hour(), clock.Minute(), clock.Second(), clock.Nanosecond(),
		clock.Location(),
	)
}

// Add returns d + duration. If duration is less than 24h (1d), then it does
// nothing.
func (d Date) Add(duration time.Duration) Date {
	return Date{t: withoutClock(d.t.Add(duration))}
}

// Eq returns true if d == other.
func (d Date) Eq(other Date) bool {
	return withoutClock(d.t).Equal(withoutClock(other.t))
}

// Before returns true if d < other.
func (d Date) Before(other Date) bool {
	return withoutClock(d.t).Before(withoutClock(other.t))
}

// After returns true if d > other.
func (d Date) After(other Date) bool {
	return withoutClock(d.t).After(withoutClock(other.t))
}

func withoutClock(t time.Time) time.Time {
	return time.Date(
		t.Year(), t.Month(), t.Day(),
		0, 0, 0, 0,
		t.Location(),
	)
}

func (d Date) String() string {
	return d.t.Format(datef)
}

func (d Date) MarshalJSON() ([]byte, error) {
	return json.Marshal(d.t.Format(datef + " MST"))
}

func (d *Date) UnmarshalJSON(b []byte) error {
	var s string
	if err := json.Unmarshal(b, &s); err != nil {
		return err
	}

	t, err := time.Parse(datef+" MST", s)
	if err != nil {
		return err
	}

	d.t = t
	return nil
}

type Scraper struct {
	Client  *http.Client
	BaseURL *url.URL
}

var baseURL, _ = url.Parse("https://gunviolencearchive.org")

// NewScraper creates a new Scraper.
func NewScraper() *Scraper {
	return &Scraper{
		Client:  http.DefaultClient,
		BaseURL: baseURL,
	}
}

var errInvalidHTML = errors.New("server returned unexpected HTML")

func selectChildIx(s *goquery.Selection, childIx int) *goquery.Selection {
	if len(s.Nodes) < childIx {
		return nil
	}
	for i := 0; i < childIx; i++ {
		s = s.Next()
	}
	return s
}

func selectText(s *goquery.Selection) string {
	if s == nil {
		return ""
	}
	return strings.TrimSpace(s.Text())
}

func selectTexts(s *goquery.Selection) []string {
	if s == nil {
		return nil
	}
	return s.Map(func(_ int, s *goquery.Selection) string {
		if s == nil {
			return ""
		}
		return strings.TrimSpace(s.Text())
	})
}

func selectHrefs(s *goquery.Selection) []string {
	if s == nil {
		return nil
	}
	return s.Map(func(_ int, s *goquery.Selection) string {
		if s == nil {
			return ""
		}
		return s.AttrOr("href", "")
	})
}

func (s *Scraper) getHTML(ctx context.Context, path string, q url.Values) (*goquery.Document, error) {
	url := *s.BaseURL
	url.Path = path
	if q != nil {
		url.RawQuery = q.Encode()
	}

	req, err := http.NewRequestWithContext(ctx, "GET", url.String(), nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/94.0.4606.71 Safari/537.36")
	if err != nil {
		return nil, errors.Wrap(err, "cannot make request")
	}

	resp, err := s.Client.Do(req)
	if err != nil {
		return nil, errors.Wrap(err, "cannot do request")
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return nil, fmt.Errorf("server returned unexpected status %s", resp.Status)
	}

	doc, err := goquery.NewDocumentFromReader(resp.Body)
	if err != nil {
		return nil, errors.Wrap(err, "cannot parse HTML")
	}

	return doc, nil
}
