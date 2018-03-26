FROM golang:latest

WORKDIR /go
COPY . .

RUN go get -d ./...
RUN go build web-service.go

EXPOSE 8080

ENTRYPOINT ["/go/web_service"]