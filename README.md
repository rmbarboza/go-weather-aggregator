# Weather Aggregator
First little project in Go

Concurrent multi-provider weather API in Go with timeouts, graceful shutdown, CI, and Docker.

## Current aggregation policy

1. Every configured provider is queried concurrently.
2. Temperature average is returned only when all providers respond successfully.
3. Any provider error invalidates all successful results.
4. The application refuses to start when no supported provider API key is configured.
5. The /weather/{city} route applies a configurable timeout and returns 504 Gateway Timeout when the deadline expires.

## Running locally

* Requirements: Go 1.26.5 .
* Configure .env file with the API KEY used for each provider. The following names need to be filled with the keys for each provider:
```
# At least one provider key is required.
OPEN_WEATHER_MAP_APPID=YOUR_WEATHER_MAP_APP_ID
WEATHER_API_KEY=YOUR_WEATHERAPI_KEY

# Optional; defaults to 3s.
WEATHER_REQUEST_TIMEOUT=3s
```
Each provider key is optional individually, but at least one provider key must be configured. When both keys are present, both providers are queried and their temperatures are averaged. The `.env` file must not be committed.
* Command to start: go run .
* URL: http://localhost:8080/weather/{city}

## Validation

```
go test ./...
go vet ./...
```

## Running in Docker

To build and run the application, you need Docker installed and running. The Dockerfile uses two stages: the first compiles the application, and the second installs CA certificates and copies the compiled binary into the runtime image.

Run the following command to generate the image:
```
docker build -t weather-aggregator:local .
```

With the generated image, just start a new container from that image. Replace YOUR_WEATHER_MAP_APP_ID with your OpenWeatherMap API key and YOUR_WEATHERAPI_KEY with WeatherAPI key. Set timeout for requests in variable WEATHER_REQUEST_TIMEOUT. This value is optional and it accepts durations like 500ms, 3s or 1m, and the default value is 3s. The -e options pass these values to the application as environment variables.
```
docker run -d --name weather-aggregator -e OPEN_WEATHER_MAP_APPID=YOUR_WEATHER_MAP_APP_ID -e WEATHER_API_KEY=YOUR_WEATHERAPI_KEY -e WEATHER_REQUEST_TIMEOUT=3s -p 8080:8080 weather-aggregator:local
```
This example uses both OPEN_WEATHER_MAP_APPID and WEATHER_API_KEY, but any one of them may be removed as long as least one is used.

Check if application is running by calling the '/health' endpoint. You should receive HTTP status 200 and the text "ok" in response.
```
curl -i http://localhost:8080/health
```

Stop the running container:
```
docker stop weather-aggregator
```

When the application receives a shutdown signal, it stops accepting new connections and waits up to 5 seconds for active requests to complete. If the timeout expires, the process exits and any remaining requests may be interrupted.

## Architecture and trade-offs

`main` loads configuration, configures structured logging, creates the weather providers and HTTP server, and coordinates graceful shutdown. `newMux` registers the HTTP routes and connects the `/weather/{city}` endpoint to the aggregator.

For each weather request, `MultiWeatherProvider` starts one goroutine per provider. The providers perform their HTTP requests concurrently and send their results through a shared channel. This reduces total latency because the application does not wait for each provider sequentially.

The results channel has capacity equal to the number of providers. If the aggregator returns early because of an error or context cancellation, a provider goroutine can still publish its result without waiting for an active receiver. The provider itself may remain blocked until its HTTP request finishes or observes context cancellation.

The current policy requires every provider to succeed. This produces a complete average with simple response semantics, but it also reduces availability because one provider failure invalidates all successful results.

The request context is propagated to every provider, allowing cancellation to reach outbound HTTP requests. The `/weather/{city}` handler derives a context with a configurable timeout from the incoming request context. Client cancellation and the application deadline are propagated to every provider. The default timeout is three seconds. When timeout is reached, it returns HTTP code 504 and a Gateway Timeout response.

Starting one goroutine per provider is simple and appropriate for the current small, fixed provider list. A dynamic or much larger provider set would require an explicit concurrency limit.
