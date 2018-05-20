package main

import "net/http"
import (
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"strconv"
	"math"
	"time"
	"encoding/json"
	"sort"
	"flag"
)

type mytype struct{}
var (
	redisClient *redis.Client
	scoreTable ScoreTable
	redisServerAddr       = flag.String("redis_server_addr", "redis:6379", "")
	)

type MessageStore struct{
	Messages []Message
}
type Message struct{
	Score uint8
	DataSource string
}
type ScoreTable struct{
	twitter []uint8
	bbc     []uint8
}
type AverageScore struct{
	createdAt time.Time
	twitter float64
	bbc		float64
}
const(
	TWITTER = "twitter"
	BBC = "bbc"
)

func (t *mytype) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there from mytype")
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	tw,bbc := buildAverage()
	//averageTwitter := Round(tw,0.05) * 100
	//averageBbc := Round(bbc,0.05) * 100

	//var mood string
	//if(average > 0.5){mood = "POSITIVE"
	//}else{mood = "NEGATIVE"}
	fmt.Fprintln(w,"SuperSentiment v2.0")
	fmt.Fprintf(w, "Twitter: %s (%.0f%%)" ,hashBarBuilder(tw),tw*100)
	fmt.Fprintln(w,"")
	fmt.Fprintf(w, "BBC News: %s (%.0f%%)",hashBarBuilder(bbc) ,bbc*100)
}
func ApiHandler(w http.ResponseWriter, r *http.Request) {
	tw,bbc := buildAverage()
	scores := AverageScore{}
	err := json.NewDecoder(r.Body).Decode(&scores)
	if err != nil{
		panic(err)
	}
	scores.twitter = tw
	scores.bbc = bbc
	scores.createdAt = time.Now().Local()

	scoreJson, err := json.Marshal(scores)
	if err != nil{
		panic(err)
	}

	w.Header().Set("Content-Type","application/json")
	w.WriteHeader(http.StatusOK)
	w.Write(scoreJson)
}

func hashBarBuilder(bar float64) string{
	cnt := 0.0
	barstr := ""
	for cnt < bar * 100{
		barstr += "#"
		cnt += 10
	}
	return barstr
}

func setupRedisClient() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     *redisServerAddr,
		Password: "", // no password set
		DB:       0,  // use default DB
	})
	snd := strconv.FormatInt(time.Now().Unix(), 10)
	err := redisClient.Set(snd,1,0 ).Err()
	if err != nil {
		panic(err)
	}
}

func (e *MessageStore) UnmarshalBinary(data []byte) error {
	if err := json.Unmarshal(data, &e); err != nil {
		return err
	}
	return nil
}

func (st *ScoreTable) getAllScores(scores MessageStore) {
	var twitterScores,bbcScores []uint8
	for _,v := range scores.Messages{
		switch v.DataSource {
			case TWITTER:{
				twitterScores = append(twitterScores, v.Score)
			}
			case BBC:{
				bbcScores = append(bbcScores,v.Score)
			}
		}
	}
	st.twitter = append(st.twitter, twitterScores...)
	st.bbc = append(st.bbc, bbcScores...)
}

func calculateTotal(scores []uint8) uint8{
	var total uint8
	for _,v := range scores {total += v}
	return total
}

func calculateAverage(total uint8,scores []uint8) float64{
	return float64(total) / float64(len(scores))
}

func getAllKeys() ([]string,error){
	return redisClient.Keys("*").Result()
}

func buildAverage() (float64,float64){
	val,err := getAllKeys()
	if err != nil {
		log.Panic(err)
	}
	sort.Strings(val)
	toInt,_ := strconv.Atoi(val[len(val)-1])
	fromInt := toInt - 60

	var ms MessageStore
	for _,v := range val {
		if fromInt < toInt {
			value, _ := redisClient.Get(strconv.Itoa(fromInt)).Result()
			ms.UnmarshalBinary([]byte(value))

			scoreTable.getAllScores(ms)
			fromInt += 1
			v=v
		}
	}
	fmt.Printf("total BBC %d total Twitter %d", len(scoreTable.bbc),len(scoreTable.twitter))

	totalTwitter := calculateTotal(scoreTable.twitter)
	totalBbc := calculateTotal(scoreTable.bbc)

	averageTwitter := calculateAverage(totalTwitter,scoreTable.twitter)
	averageBbc := calculateAverage(totalBbc,scoreTable.bbc)

	scoreTable.twitter = []uint8{}
	scoreTable.bbc = []uint8{}
	return averageTwitter,averageBbc
}

func main() {
	setupRedisClient()

	t := new(mytype)
	http.Handle("/", t)

	http.HandleFunc("/sentiment/", StatusHandler)
	http.HandleFunc("/sentiment/api/", ApiHandler)

	http.ListenAndServe(":8080", nil)
}