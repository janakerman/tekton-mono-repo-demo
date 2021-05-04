FROM golang:1.16 as builder

COPY . /src/
WORKDIR /src/
RUN CGO_ENABLED=0 go build -o /interceptor cmd/interceptor.go

FROM alpine

COPY --from=builder /interceptor /usr/local/bin/
ENTRYPOINT interceptor
