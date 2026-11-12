package api

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/bruli/raspberryWaterSystem/internal/domain/status"
)

const urlRaw = "https://api.sunrise-sunset.org/json?lat=%s&lng=%s&date=%s&formatted=0"

type results struct {
	Results struct {
		Sunrise time.Time `json:"sunrise"`
		Sunset  time.Time `json:"sunset"`
	} `json:"results"`
}

type SunriseSunsetRepository struct {
	cl                  *http.Client
	latitude, longitude string
	location            *time.Location
	cache               map[string]*status.Light
	m                   sync.RWMutex
}

func (s *SunriseSunsetRepository) Find(ctx context.Context, date time.Time) (*status.Light, error) {
	day := s.formatDay(date)
	s.m.RLock()
	light, ok := s.cache[day]
	s.m.RUnlock()
	if ok {
		return light, nil
	}
	return s.getAndCache(ctx, day)
}

func (s *SunriseSunsetRepository) CleanYesterday(ctx context.Context) {
	tick := time.NewTicker(4 * time.Hour)
	defer tick.Stop()
	for {
		select {
		case <-ctx.Done():
			return
		case <-tick.C:
			s.removeBefore(s.formatDay(time.Now()))
		}
	}
}

func (s *SunriseSunsetRepository) removeBefore(day string) {
	s.m.Lock()
	defer s.m.Unlock()
	for cached := range s.cache {
		if cached < day {
			delete(s.cache, cached)
		}
	}
}

func (s *SunriseSunsetRepository) formatDay(date time.Time) string {
	return date.In(s.location).Format(time.DateOnly)
}

func (s *SunriseSunsetRepository) getAndCache(ctx context.Context, day string) (*status.Light, error) {
	url := fmt.Sprintf(urlRaw, s.latitude, s.longitude, day)

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, http.NoBody)
	if err != nil {
		return nil, fmt.Errorf("unable to create request: %w", err)
	}
	req.Header.Set("Content-Type", "application/json")
	resp, err := s.cl.Do(req)
	if err != nil {
		return nil, fmt.Errorf("unable to get light: %w", err)
	}
	defer func() {
		_ = resp.Body.Close()
	}()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("sunrise sunset api returned status %d", resp.StatusCode)
	}
	var re results
	if err = json.NewDecoder(resp.Body).Decode(&re); err != nil {
		return nil, fmt.Errorf("failed to unmarshal response: %w", err)
	}
	li, err := status.NewLight(re.Results.Sunrise.In(s.location), re.Results.Sunset.In(s.location))
	if err != nil {
		return nil, fmt.Errorf("failed to parse sunrise light: %w", err)
	}
	s.m.Lock()
	defer s.m.Unlock()
	s.cache[day] = li
	return li, nil
}

func NewSunriseSunsetRepository(latitude, longitude float64, location *time.Location, timeout time.Duration) *SunriseSunsetRepository {
	return &SunriseSunsetRepository{
		cl:        &http.Client{Timeout: timeout},
		latitude:  strconv.FormatFloat(latitude, 'f', -1, 64),
		longitude: strconv.FormatFloat(longitude, 'f', -1, 64),
		location:  location,
		cache:     make(map[string]*status.Light),
	}
}
