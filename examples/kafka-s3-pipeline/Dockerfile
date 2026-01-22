FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY src/ ./
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /pipeline .

FROM gcr.io/distroless/static-debian12:nonroot
COPY --from=builder /pipeline /pipeline
ENTRYPOINT ["/pipeline"]
