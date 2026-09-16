package main

import (
	"context"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

import weatherapps "github.com/rmbarboza/gofirstproject/weather_apps"

func TestHealthEndpoint(t *testing.T) {
	req := httptest.NewRequest(http.MethodGet, "/health", nil)
	rec := httptest.NewRecorder()

	wantCode := http.StatusOK
	wantBody := "ok\n"

	mux := newMux(nil, 10*time.Second)
	mux.ServeHTTP(rec, req)

	if rec.Code != wantCode {
		t.Fatalf("GET /health returned code: %v; want %v", rec.Code, wantCode)
	}

	if rec.Body.String() != wantBody {
		t.Fatalf("GET /health returned body %q; want %q", rec.Body.String(), wantBody)
	}
}

type providerThatWaitsForCancellation struct{}

func (providerThatWaitsForCancellation) Temperature(ctx context.Context, _ string) (float64, error) {
	<-ctx.Done()
	return 0, ctx.Err()
}

func TestWeatherEndpointReturnsGatewayTimeout(t *testing.T) {
	mw := weatherapps.MultiWeatherProvider{
		providerThatWaitsForCancellation{},
	}

	req := httptest.NewRequest(http.MethodGet, "/weather/Recife", nil)
	rec := httptest.NewRecorder()

	wantCode := http.StatusGatewayTimeout

	mux := newMux(mw, 10*time.Millisecond)

	mux.ServeHTTP(rec, req)

	if rec.Code != wantCode {
		t.Fatalf("GET /weather/Recife returned code: %v; want %v", rec.Code, wantCode)
	}
}

func TestConfiguredProviders(t *testing.T) {
	type testCase struct {
		name                 string
		openWeatherMapAPIKey string
		weatherAPIKey        string
		wantProviderCount    int
		wantErr              bool
	}

	tests := []testCase{
		{
			name:                 "2 providers without error",
			openWeatherMapAPIKey: "test-key",
			weatherAPIKey:        "test-key",
			wantProviderCount:    2,
			wantErr:              false,
		},
		{
			name:                 "only open weather without error",
			openWeatherMapAPIKey: "test-key",
			wantProviderCount:    1,
			wantErr:              false,
		},
		{
			name:              "only weatherapi without error",
			weatherAPIKey:     "test-key",
			wantProviderCount: 1,
			wantErr:           false,
		},
		{
			name:              "0 provider with error",
			wantProviderCount: 0,
			wantErr:           true,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mw, err := configuredProviders(tc.openWeatherMapAPIKey, tc.weatherAPIKey)

			if tc.wantErr && err == nil {
				t.Error("configuredProviders() error = nil, want an error")
			}

			if !tc.wantErr && err != nil {
				t.Errorf("configuredProviders() error = %v, want nil", err)
			}

			if got := len(mw); got != tc.wantProviderCount {
				t.Errorf(
					"configuredProviders() provider count = %d, want %d",
					got,
					tc.wantProviderCount,
				)
			}
		})
	}
}
