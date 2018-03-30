package main

import (
	"fmt"
	"log"
	"os"
	"os/signal"
	"syscall"
	"github.com/dghubble/go-twitter/twitter"
	"github.com/dghubble/oauth1"
	"google.golang.org/grpc"
	pb "github.com/vilius-valiusis/twitter_app/stubs/twitter_stub"
	"net"
	"flag"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/testdata"
)

var (
	tls        = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	certFile   = flag.String("cert_file", "", "The TLS cert file")
	keyFile    = flag.String("key_file", "", "The TLS key file")
	port       = flag.Int("port", 3000, "The server port")
)

type Server struct{}

func (s *Server) GetTweets(in *pb.TweetRequest,tweetStream pb.TwitterService_GetTweetsServer) error {
	//////////////////////////////////////////////
	// Setting up the go-twitter streaming API
	//////////////////////////////////////////////
	consumerKey := "ts6qnkjoZTvD6Y8BtsM1GjpGq"
	consumerSecret := "8smEybTOYRPuM02gYXqYPygJ01ga25cQAaYpgoZ34eosKygJ5C"
	accessToken := "975696773422485504-vIKtCRGx8lrYGF0Nx4gaPuCNjek6mbA"
	accessSecret := "YY8mVqyxSG2Fgs4Jkpb6X6Naw3DaLNf6dQXtxdlU01hbA"
	// Pass in your consumer key (API Key) and your Consumer Secret (API Secret)
	config := oauth1.NewConfig(consumerKey, consumerSecret)
	// Pass in your Access Token and your Access Token Secret
	token := oauth1.NewToken(accessToken, accessSecret)
	httpClient := config.Client(oauth1.NoContext, token)
	client := twitter.NewClient(httpClient)

	demux := twitter.NewSwitchDemux()

	// FILTER based on parameters
	filterParams := &twitter.StreamFilterParams{
		Track:         []string{in.Name},
		StallWarnings: twitter.Bool(true),
	}

	// Send out a response on every new tweet
	demux.Tweet = func(tweet *twitter.Tweet){
		log.Println(tweet.Text)
		tweetStream.Send(&pb.TweetResponse{TweetText:tweet.Text})
	}

	fmt.Println("Starting Stream...")
	stream, err := client.Streams.Filter(filterParams)

	if err != nil {
		log.Fatal(err)
	}

	// Receive messages until stopped or stream quits
	go demux.HandleChan(stream.Messages)

	// Wait for SIGINT and SIGTERM (HIT CTRL-C)
	ch := make(chan os.Signal)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	log.Println(<-ch)

	fmt.Println("Stopping Stream...")
	stream.Stop()
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
	pb.RegisterTwitterServiceServer(grpcServer, &Server{})
	grpcServer.Serve(lis)
}
