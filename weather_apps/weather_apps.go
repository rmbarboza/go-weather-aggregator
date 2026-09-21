package weatherapps

import (
	"context"
	"encoding/json"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
)

type WeatherProvider interface {
	Temperature(ctx context.Context, city string) (float64, error) // in Kelvin, naturally
}

type OpenWeatherMap struct {
	APIKey  string
	BaseURL string
}

type WeatherUnderground struct {
	APIKey  string
	BaseURL string
}

type WeatherAPI struct {
	APIKey  string
	BaseURL string
}

const (
	openWeatherMapBaseURL     = "https://api.openweathermap.org"
	weatherUndergroundBaseURL = "http://api.wunderground.com"
	weatherAPIBaseURL         = "https://api.weatherapi.com"
)

// Method for open weather map
func (w OpenWeatherMap) Temperature(ctx context.Context, city string) (float64, error) {
	baseURL := w.BaseURL
	if baseURL == "" {
		baseURL = openWeatherMapBaseURL
	}

	url := baseURL + "/data/2.5/weather?APPID=" + w.APIKey + "&q=" + city

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("OpenWeatherMap returned unexpected HTTP status %s", resp.Status)
	}

	var d struct {
		Main struct {
			Kelvin float64 `json:"temp"`
		} `json:"main"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	slog.Info(
		"weather provider returned temperature",
		"provider", "openWeatherMap",
		"city", city,
		"temperature_kelvin", d.Main.Kelvin,
	)
	return d.Main.Kelvin, nil
}

// Method for weather underground
func (w WeatherUnderground) Temperature(ctx context.Context, city string) (float64, error) {
	baseURL := w.BaseURL
	if baseURL == "" {
		baseURL = weatherUndergroundBaseURL
	}

	url := baseURL + "/api/" + w.APIKey + "/conditions/q/" + city + ".json"

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, url, nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("WeatherUnderground returned unexpected HTTP status %s", resp.Status)
	}

	var d struct {
		Observation struct {
			Celsius float64 `json:"temp_c"`
		} `json:"current_observation"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	kelvin := d.Observation.Celsius + 273.15
	slog.Info(
		"weather provider returned temperature",
		"provider", "weatherUnderground",
		"city", city,
		"temperature_kelvin", kelvin,
	)
	return kelvin, nil
}

// Method for weather api
func (w WeatherAPI) Temperature(ctx context.Context, city string) (float64, error) {
	baseURL := w.BaseURL
	if baseURL == "" {
		baseURL = weatherAPIBaseURL
	}

	requestURL, err := url.Parse(baseURL)
	if err != nil {
		return 0, fmt.Errorf("parse WeatherAPI base URL: %w", err)
	}

	requestURL.Path = "/v1/current.json"

	query := requestURL.Query()
	query.Set("key", w.APIKey)
	query.Set("q", city)
	requestURL.RawQuery = query.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, requestURL.String(), nil)
	if err != nil {
		return 0, err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return 0, err
	}

	defer resp.Body.Close()

	if resp.StatusCode < http.StatusOK || resp.StatusCode >= http.StatusMultipleChoices {
		return 0, fmt.Errorf("WeatherAPI returned unexpected HTTP status %s", resp.Status)
	}

	var d struct {
		Current struct {
			Celsius float64 `json:"temp_c"`
		} `json:"current"`
	}

	if err := json.NewDecoder(resp.Body).Decode(&d); err != nil {
		return 0, err
	}

	kelvin := d.Current.Celsius + 273.15
	slog.Info(
		"weather provider returned temperature",
		"provider", "weatherAPI",
		"city", city,
		"temperature_kelvin", kelvin,
	)
	return kelvin, nil
}
