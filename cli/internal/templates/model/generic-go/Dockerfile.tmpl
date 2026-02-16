FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o /model .

FROM alpine:3.19
COPY --from=builder /model /model
ENTRYPOINT ["/model"]
