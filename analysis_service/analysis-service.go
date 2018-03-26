package main

import (
	"fmt"
	"log"
	"google.golang.org/grpc"
	pb "github.com/vilius-valiusis/twitter_app/twitter_app"
	"google.golang.org/grpc/testdata"
	"flag"
	"google.golang.org/grpc/credentials"
	"golang.org/x/net/context"
	"io"
	"github.com/cdipaolo/sentiment"
	"github.com/go-redis/redis"
	"time"
	"strconv"
)


var (
	tls                = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile             = flag.String("ca_file", "", "The file containning the CA root cert file")
	serverAddr         = flag.String("server_addr", "127.0.0.1:10000", "The server address in the format of host:port")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
	redisClient *redis.Client
)

func performSentimentAnalysis(model *sentiment.Models, res string) uint8{
	analysis := model.SentimentAnalysis(res, sentiment.English)
	return analysis.Score
}

func sentimentAnalysis(client pb.TwitterServiceClient,tweet *pb.TweetRequest ) {
	 //Restore the sentiment model
	model, err := sentiment.Restore()
	if err != nil {
		panic(fmt.Sprintf("Could not restore model!\n\t%v\n", err))
	}

	stream, err := client.GetTweets(context.Background(),tweet)
	if err != nil {
		log.Fatalf("%v.ListFeatures(_) = _, %v", client, err)
	}
	for {
		feature, err := stream.Recv()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Fatalf("%v.ListFeatures(_) = _, %v", client, err)
		}
		score := performSentimentAnalysis(&model, feature.TweetText)
		log.Println(score)
		//Seconds passed since january 1970
		snd := strconv.FormatInt(time.Now().Unix(), 10)
		err = redisClient.Set(snd,score,0 ).Err() // 10 minutes
		if err != nil {
			panic(err)
		}
	}
}

func setupRedisClient() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	redisClient.FlushAll()
}

func main() {
	setupRedisClient()
	flag.Parse()
	var opts []grpc.DialOption
	if *tls {
		if *caFile == "" {
			*caFile = testdata.Path("ca.pem")
		}
		creds, err := credentials.NewClientTLSFromFile(*caFile, *serverHostOverride)
		if err != nil {
			log.Fatalf("Failed to create TLS credentials %v", err)
		}
		opts = append(opts, grpc.WithTransportCredentials(creds))
	} else {
		opts = append(opts, grpc.WithInsecure())
	}

	conn, err := grpc.Dial(*serverAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pb.NewTwitterServiceClient(conn)
	sentimentAnalysis(client,&pb.TweetRequest{Name:"dog"})


}
