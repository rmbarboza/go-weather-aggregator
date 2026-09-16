package weatherapps

import (
	"context"
	"errors"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func noContentHandler(w http.ResponseWriter, r *http.Request) {
	w.WriteHeader(http.StatusNoContent)
}

func TestHTTPTestServerReturnsControlledStatus(t *testing.T) {

	server := httptest.NewServer(http.HandlerFunc(noContentHandler))
	defer server.Close()

	resp, err := server.Client().Get(server.URL)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent {
		t.Errorf("Expected status %d, got %d", http.StatusNoContent, resp.StatusCode)
	}
}

func TestOpenWeatherMapReturnsWhenContextIsCanceled(t *testing.T) {

	requestStarted := make(chan struct{})
	errorChan := make(chan error, 1)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		close(requestStarted)
		<-r.Context().Done()
	}))
	defer server.Close()

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	provider := OpenWeatherMap{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}

	go func() {
		_, err := provider.Temperature(ctx, "tokyo")
		errorChan <- err
	}()

	select {
	case <-requestStarted:
		// Chegou ao servidor.
	case <-time.After(time.Second):
		t.Fatal("request did not reach the test server")
	}

	cancel()

	select {
	case err := <-errorChan:
		if !errors.Is(err, context.Canceled) {
			t.Fatalf("error = %v, want %v", err, context.Canceled)
		}
	case <-time.After(time.Second):
		t.Fatal("OpenWeatherMap did not return after context cancellation")
	}
}

func TestOpenWeatherMapReturnsErrorForNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	provider := OpenWeatherMap{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}

	_, err := provider.Temperature(context.Background(), "tokyo")

	if err == nil {
		t.Fatal("error should not be nil")
	}
}

func TestWeatherUndergroundReturnsErrorForNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	provider := WeatherUnderground{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}

	_, err := provider.Temperature(context.Background(), "tokyo")

	if err == nil {
		t.Fatal("error should not be nil")
	}
}

func TestWeatherAPIReturnsTemperatureInKelvin(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if got, want := r.Method, "GET"; got != want {
			t.Errorf("invalid request method %v, want %v", got, want)
		}

		if got, want := r.URL.Path, "/v1/current.json"; got != want {
			t.Errorf("invalid request path %v, want %v", got, want)
		}

		if got, want := r.URL.Query().Get("key"), "test-key"; got != want {
			t.Errorf("invalid key %v, want %v", got, want)
		}

		if got, want := r.URL.Query().Get("q"), "São Paulo"; got != want {
			t.Errorf("invalid q %v, want %v", got, want)
		}

		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte(`{"current":{"temp_c":20.0}}`))
	}))
	defer server.Close()

	provider := WeatherAPI{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}

	wantTemp := 293.15

	temp, err := provider.Temperature(context.Background(), "São Paulo")

	if err != nil {
		t.Fatalf("request returned with error %v", err)
	}

	if temp != wantTemp {
		t.Errorf("request returned temp %f, want %f", temp, wantTemp)
	}
}

func TestWeatherAPIReturnsErrorForNonSuccessStatus(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("{}"))
	}))
	defer server.Close()

	provider := WeatherAPI{
		APIKey:  "test-key",
		BaseURL: server.URL,
	}

	_, err := provider.Temperature(context.Background(), "São Paulo")

	if err == nil {
		t.Fatal("error should not be nil")
	}
}
