FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod ./
COPY *.go ./

RUN go mod tidy \
  && go mod download \
  && CGO_ENABLED=0 GOOS=linux go build -a -ldflags '-w -s' -o demo-app .

FROM scratch

COPY --from=builder /app/demo-app /demo-app

EXPOSE 8080

CMD ["/demo-app"]
