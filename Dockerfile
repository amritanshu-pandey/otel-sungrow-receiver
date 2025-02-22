FROM golang:1.24.0 AS build
COPY . /app/
WORKDIR /app/
RUN go mod download
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -ldflags="-w -s" -o /otel-sungrow-receiver

FROM scratch
COPY --from=build /otel-sungrow-receiver /
EXPOSE 9100
ENTRYPOINT ["/otel-sungrow-receiver"] 
