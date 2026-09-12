FROM golang:1.26.5-alpine AS build

WORKDIR /src

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /out/weather-aggregator .

FROM alpine:3.23 AS runtime

RUN apk add --no-cache ca-certificates

COPY --from=build /out/weather-aggregator /usr/local/bin/weather-aggregator

EXPOSE 8080

CMD ["/usr/local/bin/weather-aggregator"]