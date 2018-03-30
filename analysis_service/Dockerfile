FROM golang:latest

WORKDIR /go
COPY . .

RUN go get -d -v ./...
RUN go build analysis-service.go


ENTRYPOINT ["/go/analysis-service"]