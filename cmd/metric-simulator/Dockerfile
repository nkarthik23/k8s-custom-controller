FROM golang:1.26 AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o metric-simulator ./cmd/metric-simulator

FROM gcr.io/distroless/static:nonroot
WORKDIR /
COPY --from=builder /app/metric-simulator .
USER 65532:65532
EXPOSE 8080
ENTRYPOINT ["/metric-simulator"]
