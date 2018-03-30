package main

import (
	"fmt"
	"log"
	"google.golang.org/grpc"
	pbt "github.com/vilius-valiusis/twitter_app/stubs/twitter_stub"
	pbb "github.com/vilius-valiusis/twitter_app/stubs/bbc_stub"
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
	twitterServerAddr  = flag.String("twitter_server_addr", "localhost:3000", "")
	bbcServerAddr  = flag.String("bbc_server_addr", "localhost:3001", "")
	serverHostOverride = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
	redisClient        *redis.Client
	sentimentModel *sentiment.Models
)

func performSentimentAnalysis(model *sentiment.Models, res string) uint8{
	analysis := model.SentimentAnalysis(res, sentiment.English)
	return analysis.Score
}

func restoreSentimentalModel(){
	model, err := sentiment.Restore()
	if err != nil {
		panic(fmt.Sprintf("Could not restore model!\n\t%v\n", err))
	}
	sentimentModel = &model
}

func twitterSentimentAnalysis(client pbt.TwitterServiceClient,tweet *pbt.TweetRequest ) {

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

		score := performSentimentAnalysis(sentimentModel, feature.TweetText)
		log.Println(score)
		//Seconds passed since january 1970
		snd := strconv.FormatInt(time.Now().Unix(), 10)
		err = redisClient.Set(snd,score,0 ).Err() // 10 minutes
		if err != nil {
			panic(err)
		}
	}
}

func newsSentimentAnalysis(client pbb.NewsServiceClient, query *pbb.NewsRequest) {

	stream, err := client.GetNews(context.Background(),query)
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

		score := performSentimentAnalysis(sentimentModel, feature.NewsText)
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

func setupTwitterClient(opts []grpc.DialOption){
	conn, err := grpc.Dial(*twitterServerAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pbt.NewTwitterServiceClient(conn)
	twitterSentimentAnalysis(client,&pbt.TweetRequest{Name:"dog"})
}

func setupBBCClient(opts []grpc.DialOption){
	conn, err := grpc.Dial(*bbcServerAddr, opts...)
	if err != nil {
		log.Fatalf("fail to dial: %v", err)
	}
	defer conn.Close()

	client := pbb.NewNewsServiceClient(conn)
	newsSentimentAnalysis(client,&pbb.NewsRequest{Query:"dog"})
}

func main() {
	setupRedisClient()
	restoreSentimentalModel()

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
	//setupTwitterClient(opts)
	setupBBCClient(opts)
}
