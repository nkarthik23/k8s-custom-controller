package main

import (
	"fmt"
	"log"
	"net/http"
	"strconv"
	"sync"
)

var (
	mu         sync.Mutex
	queueDepth float64 = 0
)

func metricsHandler(w http.ResponseWriter, r *http.Request) {
	mu.Lock()
	defer mu.Unlock()
	w.Header().Set("Content-Type", "text/plain; version=0.0.4")
	fmt.Fprintf(w, "# HELP queue_depth Number of items waiting in the queue\n")
	fmt.Fprintf(w, "# TYPE queue_depth gauge\n")
	fmt.Fprintf(w, "queue_depth %f\n", queueDepth)
}

func setHandler(w http.ResponseWriter, r *http.Request) {
	valStr := r.URL.Query().Get("value")
	val, err := strconv.ParseFloat(valStr, 64)
	if err != nil {
		http.Error(w, "invalid value", http.StatusBadRequest)
		return
	}
	mu.Lock()
	queueDepth = val
	mu.Unlock()
	fmt.Fprintf(w, "queue_depth set to %f\n", val)
}

func main() {
	http.HandleFunc("/metrics", metricsHandler)
	http.HandleFunc("/set", setHandler)
	log.Println("metric-simulator listening on :8080")
	log.Fatal(http.ListenAndServe(":8080", nil))
}
