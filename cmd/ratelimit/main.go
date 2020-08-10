package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Bucket represents the token bucket for a specific client.
type Bucket struct {
	Value        int
	MaxAmount    int
	LastUpdate   time.Time
	RefillTime   int // seconds
	RefillAmount int
}

const maxAmount int = 5

func main() {
	buckets := make(map[string]Bucket)
	var m sync.Mutex

	http.HandleFunc("/register_key", registerKeyHandler(buckets, &m))
	http.HandleFunc("/use_token", useTokenHandler(buckets, &m))
	log.Fatal(http.ListenAndServe(":8001", nil))
}

func registerKeyHandler(buckets map[string]Bucket, m *sync.Mutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		id := uuid.New()
		m.Lock()
		buckets[id.String()] = Bucket{
			MaxAmount:    maxAmount,
			Value:        maxAmount,
			LastUpdate:   time.Now(),
			RefillTime:   5,
			RefillAmount: 1,
		}.refillBucket()
		m.Unlock()

		fmt.Fprint(w, id.String())
	}
}

func useTokenHandler(buckets map[string]Bucket, m *sync.Mutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		u := r.URL.Query().Get("uuid")
		b, ok := buckets[u]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "didn't find bucket")
			return
		}
		b, notlimited := b.reduce()
		if !notlimited {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, "slow down there, turbo")
			return
		}

		m.Lock()
		buckets[u] = b
		m.Unlock()
		fmt.Fprintf(w, "go ahead and do your thing. you have %d tokens left.", b.Value)
	}
}

func (b Bucket) refillBucket() Bucket {
	refillCount := int(math.Floor(time.Now().Sub(b.LastUpdate).Seconds() / float64(b.RefillTime)))

	b.Value += refillCount * b.RefillAmount
	if b.Value > b.MaxAmount {
		b.Value = b.MaxAmount
	}

	b.LastUpdate = b.LastUpdate.Add(time.Duration(refillCount*b.RefillTime) * time.Second)
	if time.Now().Before(b.LastUpdate) {
		b.LastUpdate = time.Now()
	}

	return b
}

func (b Bucket) reduce() (Bucket, bool) {
	b = b.refillBucket()
	if b.Value == 0 {
		return b, false
	}
	b.Value--
	return b, true
}
