// Package clients · OpenWeatherMap — proxies daily weather for IMD-grade
// coverage. We cache results in the weather_snapshots table for 15 min.
package clients

import (
	"context"
	"encoding/json"
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
// If no API key is configured, falls back to Open-Meteo (free, no key).
func (o *OpenWeather) Fetch(ctx context.Context, lat, lng float64) (*CurrentWeather, error) {
	if o.APIKey == "" {
		return fetchOpenMeteo(ctx, o.HTTP, lat, lng)
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

// fetchOpenMeteo — free, no-key weather fallback. Used when
// OPENWEATHERMAP_API_KEY is empty. Endpoint: api.open-meteo.com.
func fetchOpenMeteo(ctx context.Context, hc *http.Client, lat, lng float64) (*CurrentWeather, error) {
	url := fmt.Sprintf(
		"https://api.open-meteo.com/v1/forecast?latitude=%f&longitude=%f"+
			"&current=temperature_2m,apparent_temperature,relative_humidity_2m,"+
			"weather_code,wind_speed_10m,precipitation"+
			"&daily=sunrise,sunset&timezone=auto",
		lat, lng,
	)
	req, _ := http.NewRequestWithContext(ctx, "GET", url, nil)
	res, err := hc.Do(req)
	if err != nil {
		return nil, fmt.Errorf("open-meteo: %w", err)
	}
	defer res.Body.Close()
	if res.StatusCode >= 400 {
		return nil, fmt.Errorf("open-meteo %d", res.StatusCode)
	}
	var raw struct {
		Current struct {
			Temp         float64 `json:"temperature_2m"`
			Apparent     float64 `json:"apparent_temperature"`
			Humidity     int     `json:"relative_humidity_2m"`
			WeatherCode  int     `json:"weather_code"`
			WindKmh      float64 `json:"wind_speed_10m"`
			Precip       float64 `json:"precipitation"`
		} `json:"current"`
		Daily struct {
			Sunrise []string `json:"sunrise"`
			Sunset  []string `json:"sunset"`
		} `json:"daily"`
	}
	if err := json.NewDecoder(res.Body).Decode(&raw); err != nil {
		return nil, err
	}

	out := &CurrentWeather{
		TempC:       raw.Current.Temp,
		FeelsLikeC:  raw.Current.Apparent,
		HumidityPct: raw.Current.Humidity,
		WindKmh:     raw.Current.WindKmh,
		PrecipMm:    raw.Current.Precip,
		Condition:   wmoCondition(raw.Current.WeatherCode),
		Icon:        wmoIcon(raw.Current.WeatherCode),
	}
	if len(raw.Daily.Sunrise) > 0 {
		if t, err := time.Parse("2006-01-02T15:04", raw.Daily.Sunrise[0]); err == nil {
			out.Sunrise = t
		}
	}
	if len(raw.Daily.Sunset) > 0 {
		if t, err := time.Parse("2006-01-02T15:04", raw.Daily.Sunset[0]); err == nil {
			out.Sunset = t
		}
	}
	return out, nil
}

// wmoCondition — WMO weather code → human label (see open-meteo docs).
func wmoCondition(code int) string {
	switch {
	case code == 0:                          return "Clear"
	case code >= 1 && code <= 3:             return "Clouds"
	case code == 45 || code == 48:           return "Fog"
	case code >= 51 && code <= 57:           return "Drizzle"
	case code >= 61 && code <= 67:           return "Rain"
	case code >= 71 && code <= 77:           return "Snow"
	case code >= 80 && code <= 82:           return "Rain"
	case code >= 85 && code <= 86:           return "Snow"
	case code >= 95 && code <= 99:           return "Thunderstorm"
	default:                                 return "Clouds"
	}
}

func wmoIcon(code int) string {
	switch {
	case code == 0:               return "01d"
	case code >= 1 && code <= 3:  return "03d"
	case code == 45 || code == 48: return "50d"
	case code >= 51 && code <= 67: return "10d"
	case code >= 71 && code <= 77: return "13d"
	case code >= 80 && code <= 82: return "09d"
	case code >= 95:               return "11d"
	default:                       return "04d"
	}
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
