// Package weather — IMD/OpenWeatherMap proxy with 15-min cache.
package weather

import (
	"net/http"
	"time"

	"github.com/go-chi/chi/v5"
	"github.com/jackc/pgx/v5/pgxpool"
	"github.com/kashmir-explorer/api/internal/clients"
	"github.com/kashmir-explorer/api/pkg/response"
)

type Service struct {
	pool *pgxpool.Pool
	ow   *clients.OpenWeather
}

func NewService(pool *pgxpool.Pool, key string) *Service {
	return &Service{pool: pool, ow: clients.NewOpenWeather(key)}
}

// GET /v1/weather/destination/{slug}
func (s *Service) ForDestination(w http.ResponseWriter, r *http.Request) {
	slug := chi.URLParam(r, "slug")

	var lng, lat float64
	var destID string
	if err := s.pool.QueryRow(r.Context(), `
		SELECT id::text, ST_X(location::geometry), ST_Y(location::geometry)
		FROM destinations WHERE slug = $1
	`, slug).Scan(&destID, &lng, &lat); err != nil {
		response.NotFound(w, "destination not found")
		return
	}

	// Cache check (15-min freshness).
	var temp, feel float64
	var cond string
	var aqi int
	cached := s.pool.QueryRow(r.Context(), `
		SELECT temp_c, feels_like_c, condition, COALESCE(aqi, 0)
		FROM weather_snapshots
		WHERE destination_id = $1::uuid AND fetched_at > now() - INTERVAL '15 minutes'
		ORDER BY fetched_at DESC LIMIT 1
	`, destID).Scan(&temp, &feel, &cond, &aqi) == nil

	if cached {
		response.OK(w, map[string]any{
			"slug": slug, "temp_c": temp, "feels_like_c": feel,
			"condition": cond, "aqi": aqi, "cached": true, "source": "cache",
		})
		return
	}

	// Live fetch from OpenWeatherMap.
	wx, err := s.ow.Fetch(r.Context(), lat, lng)
	if err != nil {
		response.Internal(w, err)
		return
	}

	// Persist snapshot.
	_, _ = s.pool.Exec(r.Context(), `
		INSERT INTO weather_snapshots
		  (destination_id, fetched_at, temp_c, feels_like_c, condition, icon,
		   wind_kmh, humidity_pct, visibility_km, aqi, precip_mm, sunrise, sunset, source)
		VALUES ($1, now(), $2, $3, $4, $5, $6, $7, $8, $9, $10, $11, $12, 'openweathermap')
	`, destID, wx.TempC, wx.FeelsLikeC, wx.Condition, wx.Icon,
		wx.WindKmh, wx.HumidityPct, wx.VisibilityKm, wx.AQI, wx.PrecipMm,
		wx.Sunrise.Format(time.TimeOnly), wx.Sunset.Format(time.TimeOnly))

	response.OK(w, map[string]any{
		"slug":         slug,
		"lat":          lat,
		"lng":          lng,
		"temp_c":       wx.TempC,
		"feels_like_c": wx.FeelsLikeC,
		"condition":    wx.Condition,
		"wind_kmh":     wx.WindKmh,
		"humidity_pct": wx.HumidityPct,
		"aqi":          wx.AQI,
		"precip_mm":    wx.PrecipMm,
		"sunrise":      wx.Sunrise.Format(time.TimeOnly),
		"sunset":       wx.Sunset.Format(time.TimeOnly),
		"cached":       false,
		"source":       "openweathermap",
	})
}
