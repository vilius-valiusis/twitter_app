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
	"encoding/json"
	"sync"
)


var (
	tls                 = flag.Bool("tls", false, "Connection uses TLS if true, else plain TCP")
	caFile              = flag.String("ca_file", "", "The file containning the CA root cert file")

	twitterServerAddr   = flag.String("twitter_server_addr", "twitter-service:3000", "")
	bbcServerAddr       = flag.String("bbc_server_addr", "bbc-service:3001", "")
	redisServerAddr       = flag.String("redis_server_addr", "redis:6379", "")

	serverHostOverride  = flag.String("server_host_override", "x.test.youtube.com", "The server name use to verify the hostname returned by TLS handshake")
	redisClient        *redis.Client
	sentimentModel     *sentiment.Models
	messages           MessageStore
	previousTime       int64 = 0
)
const(
	EXPIRE_TIME = 3 * time.Minute
	TWITTER = "twitter"
	BBC = "bbc"
)
type MessageStore struct{
	Messages []Message
}
type Message struct{
	Score uint8
	DataSource string
}

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

func (m MessageStore) MarshalBinary() ([]byte, error) {
	return json.Marshal(m)
}

func appendMessage(source string,submitScore uint8){
	var m = Message{}
	switch source{
		case BBC:
			m = Message{Score : submitScore,DataSource : BBC}

		case TWITTER:
			m = Message{Score : submitScore,DataSource : TWITTER}
	}
	messages.Messages = append(messages.Messages, m)
}

func submitResults(submitTime int64,submitScore uint8,source string){

	if submitTime == previousTime || previousTime == 0{
		appendMessage(source,submitScore)
		previousTime = submitTime
	}else{
		previousTime = submitTime
		fmt.Println(messages)
		seconds := strconv.FormatInt(submitTime, 10)
		err := redisClient.Set(seconds, messages, EXPIRE_TIME).Err()
		if err != nil {
			panic(err)
		}

		messages.Messages = []Message{}
		appendMessage(source,submitScore)
	}
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
		submitResults(time.Now().Unix(),score,TWITTER)
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
		submitResults(time.Now().Unix(),score,BBC)
	}
}

func setupRedisClient() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     *redisServerAddr,
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

func setupBBCClient(opts []grpc.DialOption) {
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
	var wg sync.WaitGroup
	wg.Add(2)

	go func() {
		setupTwitterClient(opts)
		wg.Done()
	}()
	go func() {
		setupBBCClient(opts)
		wg.Done()
	}()

	wg.Wait()
}
