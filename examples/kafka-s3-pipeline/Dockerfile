FROM golang:1.25-alpine AS builder

WORKDIR /app
COPY src/ ./
RUN go build -o /pipeline .

FROM alpine:3.19
COPY --from=builder /pipeline /pipeline
ENTRYPOINT ["/pipeline"]
