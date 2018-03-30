FROM golang:latest

WORKDIR /go
COPY . .

RUN go get -d -v ./...
RUN go build twitter-service.go

EXPOSE 3000

ENTRYPOINT ["/go/twitter-service"]