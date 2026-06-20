FROM golang:1.26.1-alpine AS builder

RUN apk add --no-cache git ca-certificates

WORKDIR /app

COPY go.mod ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build \ 
  -trimpath \
  -ldflags="-s -w" \
  -o app

FROM gcr.io/distroless/static-debian12:nonroot

COPY --from=builder --chown=65532:65532 /app/app . 

EXPOSE 8080

CMD ["./app"]

