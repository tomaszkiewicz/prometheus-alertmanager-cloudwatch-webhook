FROM golang:1.11-alpine as builder

RUN apk update \
 && apk add git ca-certificates

RUN adduser --uid 10000 -D -g '' user

COPY . /src
WORKDIR /src

RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -a -installsuffix cgo -ldflags="-w -s" -o app ./cmd/webhook

FROM scratch

COPY --from=builder /etc/ssl/certs/ca-certificates.crt /etc/ssl/certs/
COPY --from=builder /etc/passwd /etc/passwd

COPY --from=builder /src/app /app

USER user
ENTRYPOINT ["/app"]