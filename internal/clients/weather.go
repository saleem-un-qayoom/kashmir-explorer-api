// Package clients · OpenWeatherMap — proxies daily weather for IMD-grade
// coverage. We cache results in the weather_snapshots table for 15 min.
package clients

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"
)

type OpenWeather struct {
	APIKey string
	HTTP   *http.Client
}

func NewOpenWeather(key string) *OpenWeather {
	return &OpenWeather{APIKey: key, HTTP: &http.Client{Timeout: 8 * time.Second}}
}

type CurrentWeather struct {
	TempC        float64
	FeelsLikeC   float64
	Condition    string
	Icon         string
	WindKmh      float64
	HumidityPct  int
	VisibilityKm float64
	AQI          int
	PrecipMm     float64
	Sunrise      time.Time
	Sunset       time.Time
}

// Fetch by lat/lng — OpenWeatherMap "current weather" + Air Pollution.
func (o *OpenWeather) Fetch(ctx context.Context, lat, lng float64) (*CurrentWeather, error) {
	if o.APIKey == "" {
		return nil, errors.New("OPENWEATHERMAP_API_KEY not set")
	}

	url := fmt.Sprintf("https://api.openweathermap.org/data/2.5/weather?lat=%f&lon=%f&units=metric&appid=%s", lat, lng, o.APIKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := o.HTTP.Do(req)
	if err != nil { return nil, err }
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("openweathermap %d", res.StatusCode)
	}
	var raw struct {
		Main struct {
			Temp      float64 `json:"temp"`
			FeelsLike float64 `json:"feels_like"`
			Humidity  int     `json:"humidity"`
		} `json:"main"`
		Weather []struct {
			Main string `json:"main"`
			Icon string `json:"icon"`
		} `json:"weather"`
		Wind struct {
			Speed float64 `json:"speed"` // m/s
		} `json:"wind"`
		Visibility int `json:"visibility"` // metres
		Sys struct {
			Sunrise int64 `json:"sunrise"`
			Sunset  int64 `json:"sunset"`
		} `json:"sys"`
		Rain map[string]float64 `json:"rain"`
		Snow map[string]float64 `json:"snow"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	out := &CurrentWeather{
		TempC:        raw.Main.Temp,
		FeelsLikeC:   raw.Main.FeelsLike,
		HumidityPct:  raw.Main.Humidity,
		WindKmh:      raw.Wind.Speed * 3.6,
		VisibilityKm: float64(raw.Visibility) / 1000,
		Sunrise:      time.Unix(raw.Sys.Sunrise, 0),
		Sunset:       time.Unix(raw.Sys.Sunset, 0),
	}
	if len(raw.Weather) > 0 {
		out.Condition = raw.Weather[0].Main
		out.Icon      = raw.Weather[0].Icon
	}
	if r, ok := raw.Rain["1h"]; ok { out.PrecipMm = r }
	if s, ok := raw.Snow["1h"]; ok { out.PrecipMm = s }

	// AQI from a parallel call.
	out.AQI = o.fetchAQI(ctx, lat, lng)
	return out, nil
}

func (o *OpenWeather) fetchAQI(ctx context.Context, lat, lng float64) int {
	url := fmt.Sprintf("http://api.openweathermap.org/data/2.5/air_pollution?lat=%f&lon=%f&appid=%s", lat, lng, o.APIKey)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := o.HTTP.Do(req)
	if err != nil { return 0 }
	defer res.Body.Close()
	var raw struct {
		List []struct {
			Main struct {
				AQI int `json:"aqi"` // 1..5
			} `json:"main"`
			Components map[string]float64 `json:"components"`
		} `json:"list"`
	}
	if json.NewDecoder(res.Body).Decode(&raw) != nil || len(raw.List) == 0 {
		return 0
	}
	// OpenWeatherMap returns 1..5; convert to a US-AQI-ish scale.
	scale := map[int]int{1: 25, 2: 75, 3: 125, 4: 175, 5: 250}
	return scale[raw.List[0].Main.AQI]
}
