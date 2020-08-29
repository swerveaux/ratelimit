package main

import (
	"fmt"
	"log"
	"math"
	"net/http"
	"strings"
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

const (
	maxAmount    int = 5
	refillTime   int = 5
	refillAmount int = 1
)

// Logger is an interface that has a single method, Log(string).
type Logger interface {
	Log(msg string)
}

// SliceLogger implements the Logger interface and saves each log entry in a slice of strings
// for later retrieval.
type SliceLogger struct {
	Logs []string
}

// Log implements the Logger interface. It just appends the message to a slice of messages it
// stores internally.
func (s *SliceLogger) Log(msg string) {
	s.Logs = append(s.Logs, msg)
}

// WrapLogs takes a page title and an arbitrary number of strings and returns a full
// HTML page with the logs in a bulleted list and then each line of msgs separated by a
// line break.
func (s *SliceLogger) WrapLogs(pageTitle string, msgs ...string) string {
	var b strings.Builder
	fmt.Fprintf(&b, "<html><head><title>%s</title></head><body><ul>\n", pageTitle)
	for i := range s.Logs {
		fmt.Fprintf(&b, "<li>%s</li>", s.Logs[i])
	}
	fmt.Fprintln(&b, "</ul><br />")
	for i := range msgs {
		fmt.Fprintf(&b, "%s<br/>", msgs[i])
	}
	fmt.Fprintln(&b, "</body></html>")
	return b.String()
}

func main() {
	buckets := make(map[string]Bucket)
	var m sync.Mutex

	http.HandleFunc("/register_key", registerKeyHandler(buckets, &m))
	http.HandleFunc("/use_token", useTokenHandler(buckets, &m))
	log.Fatal(http.ListenAndServe(":8001", nil))
}

func registerKeyHandler(buckets map[string]Bucket, m *sync.Mutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Initialize new logger that just writes to a slice of strings for later repeating.
		logger := &SliceLogger{}
		id := uuid.New()
		now := time.Now()
		m.Lock()
		buckets[id.String()] = Bucket{
			MaxAmount:    maxAmount,
			Value:        maxAmount,
			LastUpdate:   now,
			RefillTime:   refillTime,
			RefillAmount: refillAmount,
		}
		m.Unlock()

		logger.Log(fmt.Sprintf("Initialized bucket with last updated time of %s and %d tokens, which is the maximum", prettyTime(now), maxAmount))
		fmt.Fprintln(w, logger.WrapLogs("Get Key", fmt.Sprintf("<a href=\"/use_token?uuid=%s\">%s</a>", id.String(), id.String())))
	}
}

func useTokenHandler(buckets map[string]Bucket, m *sync.Mutex) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		// Normally I wouldn't initialize a new logger for each request, but since this one keeps
		// state for a specific request I will.
		logger := &SliceLogger{}
		u := r.URL.Query().Get("uuid")
		b, ok := buckets[u]
		if !ok {
			w.WriteHeader(http.StatusNotFound)
			fmt.Fprintf(w, "didn't find bucket")
			return
		}
		logger.Log("Trying to take a token...")
		b, notLimited := b.reduce(logger)
		if !notLimited {
			w.WriteHeader(http.StatusTooManyRequests)
			fmt.Fprintf(w, logger.WrapLogs("Rate Limited", "slow down there, turbo"))
			return
		}

		m.Lock()
		buckets[u] = b
		m.Unlock()
		fmt.Fprintln(w, logger.WrapLogs("Using a token", fmt.Sprintf("go ahead and do your thing. you have %d tokens left.", b.Value)))
	}
}

func (b Bucket) refillBucket(logger Logger) Bucket {
	timePassed := time.Now().Sub(b.LastUpdate).Seconds()
	logger.Log(fmt.Sprintf("%f seconds have passed since the last time a token was added to the bucket.", timePassed))
	refillTimesF := timePassed / float64(b.RefillTime)

	refillTimesFloored := math.Floor(refillTimesF)
	refillTimes := int(refillTimesFloored)
	logger.Log(fmt.Sprintf("In that time, the bucket would have been refilled %d times (%f seconds passed since last update divided by update interval of %d seconds = %f, floored to %f)", refillTimes, timePassed, b.RefillTime, refillTimesF, refillTimesFloored))
	refillCount := refillTimes

	oldValue := b.Value
	b.Value += refillCount * b.RefillAmount
	logger.Log(fmt.Sprintf("Refilling the refill amount (%d) of tokens %d times to the existing value of %d leaves us with %d tokens in the bucket", b.RefillAmount, refillCount, oldValue, b.Value))
	if b.Value > b.MaxAmount {
		logger.Log(fmt.Sprintf("%d tokens is above the maximum allowed, so resetting to %d", b.Value, b.MaxAmount))
		b.Value = b.MaxAmount
	}

	logger.Log(fmt.Sprintf("Adding %d seconds to existing refill time of %s", refillCount*b.RefillTime, prettyTime(b.LastUpdate)))
	b.LastUpdate = b.LastUpdate.Add(time.Duration(refillCount*b.RefillTime) * time.Second)
	if time.Now().Before(b.LastUpdate) {
		logger.Log("Last updated would be in the future, resetting to the current time.")
		b.LastUpdate = time.Now()
	}

	return b
}

func (b Bucket) reduce(logger Logger) (Bucket, bool) {
	logger.Log("Refilling the bucket if necessary...")
	b = b.refillBucket(logger)
	if b.Value == 0 {
		logger.Log("Bucket is empty, returning false")
		return b, false
	}
	logger.Log(fmt.Sprintf("Bucket is not empty, taking 1 token from current amount of %d tokens, leaving %d", b.Value, b.Value-1))
	b.Value--
	return b, true
}

func prettyTime(t time.Time) string {
	return t.Format("2006-01-02 15:04:05")
}
