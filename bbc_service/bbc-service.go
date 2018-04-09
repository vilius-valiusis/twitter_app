package main

import (
	"fmt"
	"log"
	"google.golang.org/grpc"
	pb "github.com/vilius-valiusis/twitter_app/stubs/bbc_stub"
	"net"
	"flag"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"
	"io/ioutil"
	"encoding/json"
	"net/http"
	"strconv"
	"time"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	port       = flag.Int("port", 3001, "The server port")
)

const (
	apiKey = "&apiKey=b177f2e17a184597b5f1ad36133436b5";
	apiUrl = "https://newsapi.org/v2/everything?q="
	apiPage = "&page="
	apiSource = "sources=bbc-news&"
	maxPageLimit = 500
)

type Source struct{
	Status string `json:status`
	TotalResults int `json:totalResults`
	Articles []Articles `json:articles`
}

type Articles struct{
	Description string `json:description`
}

type Server struct{}

func (s *Server) GetNews(in *pb.NewsRequest,stream pb.NewsService_GetNewsServer) error {
	// Do a GET request to the api
	var currentPage = 0
	for currentPage < maxPageLimit{
		time.Sleep(3 * time.Second)
		response, err := http.Get(apiUrl + in.Query + apiPage + apiSource + strconv.Itoa(currentPage) + apiKey)

		if err != nil {
			fmt.Printf("The HTTP request failed with error %s\n", err)
		}

		responseData, err := ioutil.ReadAll(response.Body)

		if err != nil {
			log.Fatal(err)
		}

		var sourceObject Source
		json.Unmarshal(responseData, &sourceObject)


		for _,v := range sourceObject.Articles{
			time.Sleep(2 * time.Millisecond)
			stream.Send(&pb.NewsResponse{NewsText: v.Description})
			log.Println(v.Description)
		}
		currentPage++
	}
	return nil
}


func main() {
	flag.Parse()
	lis, err := net.Listen("tcp", fmt.Sprintf(":%d", *port))
	if err != nil {
		log.Fatalf("failed to listen: %v", err)
	}
	var opts []grpc.ServerOption
	if *tls {
		if *certFile == "" {
			*certFile = testdata.Path("server1.pem")
		}
		if *keyFile == "" {
			*keyFile = testdata.Path("server1.key")
		}
		creds, err := credentials.NewServerTLSFromFile(*certFile, *keyFile)
		if err != nil {
			log.Fatalf("Failed to generate credentials %v", err)
		}
		opts = []grpc.ServerOption{grpc.Creds(creds)}
	}
	grpcServer := grpc.NewServer(opts...)
	pb.RegisterNewsServiceServer(grpcServer, &Server{})
	grpcServer.Serve(lis)
}
