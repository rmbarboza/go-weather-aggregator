package main

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
)

import weatherapps "github.com/rmbarboza/go-weather-aggregator/weather_apps"

func configuredProviders(openWeatherMapAPIKey string, weatherAPIKey string) (weatherapps.MultiWeatherProvider, error) {
	var providers weatherapps.MultiWeatherProvider

	if openWeatherMapAPIKey != "" {
		providers = append(providers, weatherapps.OpenWeatherMap{APIKey: openWeatherMapAPIKey})
	}

	if weatherAPIKey != "" {
		providers = append(providers, weatherapps.WeatherAPI{APIKey: weatherAPIKey})
	}

	if len(providers) == 0 {
		return nil, errors.New("no weather provider API key configured")
	}

	return providers, nil
}

func hello(w http.ResponseWriter, r *http.Request) {
	w.Write([]byte("hello!"))
}

func health(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("ok\n"))
}

func newMux(mw weatherapps.MultiWeatherProvider, requestTimeout time.Duration) *http.ServeMux {
	mux := http.NewServeMux()

	mux.HandleFunc("/hello", hello)

	mux.HandleFunc("/health", health)

	mux.HandleFunc("/weather/", func(w http.ResponseWriter, r *http.Request) {
		begin := time.Now()
		city := strings.SplitN(r.URL.Path, "/", 3)[2]

		ctx, cancel := context.WithTimeout(r.Context(), requestTimeout)
		defer cancel()

		temp, err := mw.Temperature(ctx, city)
		if err != nil {
			if errors.Is(err, context.DeadlineExceeded) {
				http.Error(w, "weather request timed out", http.StatusGatewayTimeout)
			} else {
				http.Error(w, "failed to get weather", http.StatusInternalServerError)
			}
			return
		}

		w.Header().Set("Content-Type", "application/json; charset=utf-8")
		json.NewEncoder(w).Encode(map[string]interface{}{
			"city": city,
			"temp": temp,
			"took": time.Since(begin).String(),
		})
	})

	return mux
}

func main() {
	logger := slog.New(slog.NewJSONHandler(os.Stdout, nil))
	slog.SetDefault(logger)

	enverr := godotenv.Load()
	if enverr != nil {
		slog.Info(
			"No .env file found, reading from system env",
			"error", enverr,
		)
	}

	// Read the keys
	openWeatherMapAPIKey := os.Getenv("OPEN_WEATHER_MAP_APPID")
	weatherAPIKey := os.Getenv("WEATHER_API_KEY")

	requestTimeoutStr := os.Getenv("WEATHER_REQUEST_TIMEOUT")
	if requestTimeoutStr == "" {
		requestTimeoutStr = "3s"
	}

	requestTimeout, err := time.ParseDuration(requestTimeoutStr)
	if err != nil {
		slog.Error(
			"Error parsing request timeout",
			"error", err,
		)
		os.Exit(1)
	}

	if requestTimeout <= 0 {
		slog.Error(
			"WEATHER_REQUEST_TIMEOUT must be greater than zero",
			"variable", "WEATHER_REQUEST_TIMEOUT",
			"value", requestTimeoutStr,
		)
		os.Exit(1)
	}

	mw, err := configuredProviders(openWeatherMapAPIKey, weatherAPIKey)
	if err != nil {
		slog.Error(
			"Error building providers",
			"error", err,
		)
		os.Exit(1)
	}

	slog.Info("weather providers configured", "count", len(mw))

	mux := newMux(mw, requestTimeout)

	signalCtx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	server := http.Server{
		Addr:    ":8080",
		Handler: mux,
	}

	serverErr := make(chan error, 1)

	go func() {
		serverErr <- server.ListenAndServe()
	}()

	select {
	case err := <-serverErr:
		if !errors.Is(err, http.ErrServerClosed) {
			slog.Error(
				"Unexpected server shutdown",
				"error", err,
			)
			os.Exit(1)
		}
	case <-signalCtx.Done():
		slog.Info("shutdown requested")
		shutdownCtx, cancel := context.WithTimeout(
			context.Background(),
			5*time.Second,
		)
		defer cancel()

		begin := time.Now()
		err := server.Shutdown(shutdownCtx)

		if err == nil {
			slog.Info(
				"Shutdown finished",
				"duration_ms", time.Since(begin).Milliseconds(),
			)
		} else {
			slog.Error(
				"Shutdown failed",
				"duration_ms", time.Since(begin).Milliseconds(),
				"error", err,
			)
		}
	}
}
