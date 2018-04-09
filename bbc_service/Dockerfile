FROM golang:latest

WORKDIR /go
COPY . .

RUN go get -d -v ./...
RUN go build bbc-service.go


ENTRYPOINT ["/go/bbc-service"]