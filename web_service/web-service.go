package main

import "net/http"
import (
	"fmt"
	"github.com/go-redis/redis"
	"log"
	"strconv"
	"math"
)

type mytype struct{}
var (redisClient *redis.Client)


func (t *mytype) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello there from mytype")
}

func Round(x, unit float64) float64 {
	return math.Round(x/unit) * unit
}

func StatusHandler(w http.ResponseWriter, r *http.Request) {
	average := Round(calculateAverage(),0.05)
	var mood string
	if(average > 0.5){mood = "POSITIVE"
	}else{mood = "NEGATIVE"}

	fmt.Fprintf(w, "Average score: %v is %s " ,average,mood)
}

func setupRedisClient() {
	redisClient = redis.NewClient(&redis.Options{
		Addr:     "localhost:6379",
		Password: "", // no password set
		DB:       0,  // use default DB
	})

}

func calculateAverage() float64{
	val,err := redisClient.Keys("*").Result()
	if err != nil {
		log.Panic(err)
	}
	toInt,_ := strconv.Atoi(val[len(val)-1])
	fromInt := toInt - 60
	var a []int
	for _,v := range val {

		va,_ := strconv.Atoi(v)

		if(fromInt < toInt){
			value,_ := redisClient.Get(strconv.Itoa(fromInt)).Result()
			valueInt,_ := strconv.Atoi(value)
			a = append(a, valueInt)
			fromInt += 1

		}else if(va < fromInt){
			fromInt += 1
		}
	}
	var total int = 0
	for _,v := range a {total += v}
	average := float64(total) / float64(len(a))
	log.Println(average,total)
	return average
}

func main() {
	setupRedisClient()

	t := new(mytype)
	http.Handle("/", t)

	http.HandleFunc("/sentiment/", StatusHandler)

	http.ListenAndServe(":8080", nil)
}