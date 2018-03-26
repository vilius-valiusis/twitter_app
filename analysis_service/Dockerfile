FROM golang:latest

WORKDIR /go
COPY . .

RUN go get -d ./...
RUN go build analysis-service.go

EXPOSE 8080

ENTRYPOINT ["/go/analysis_service"]