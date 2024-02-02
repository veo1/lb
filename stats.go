package main

import (
	"log"
	"sync/atomic"
	"time"

	"gopkg.in/natefinch/lumberjack.v2"
)

type Stats struct {
	RequestCount int64
	ErrorCount   int64
	Latency      int64
}

func (b *Backend) IncrementRequestCount() {
	atomic.AddInt64(&b.Stats.RequestCount, 1)
}

func (b *Backend) IncrementErrorCount() {
	atomic.AddInt64(&b.Stats.ErrorCount, 1)
}

func (b *Backend) AddLatency(latency time.Duration) {
	atomic.AddInt64(&b.Stats.Latency, int64(latency.Nanoseconds()))
}

func WriteStatsToFile() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	log.SetOutput(&lumberjack.Logger{
		Filename:   "stats.txt", // Filename is the file to write logs to
		MaxSize:    10,          // MaxSize is the maximum size in megabytes before it gets rotated
		MaxBackups: 3,           // MaxBackups is the maximum number of old log files to retain
		MaxAge:     28,          // MaxAge is the maximum number of days to retain old log files
		Compress:   true,        // Compress indicates if the rotated log files should be compressed using gzip
	})

	for range ticker.C {
		for _, b := range serverPool.backends {
			stats := b.Stats
			requestCount := atomic.LoadInt64(&stats.RequestCount)
			errorCount := atomic.LoadInt64(&stats.ErrorCount)
			latency := atomic.LoadInt64(&stats.Latency)
			log.Printf("Backend URL: %s, Requests served: %d, Errors: %d, Total Latency: %d\n", b.URL.String(), requestCount, errorCount, latency)
		}
	}
}
