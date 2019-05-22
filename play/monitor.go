package play

import (
	"context"
	"fmt"
	"math"
	"net/http"
	_ "net/http/pprof"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const monitoringFrequency = 2 * time.Second

var (
	successfulRequests  uint64
	failedRequests      uint64
	discardedLogLines   uint64
	lastMeasurementTime time.Time
	lastRequestCount    uint64
	latencySinceEpochMs uint64
)

// Loops forever and emits stats every frequency duration
func monitor(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
	lastMeasurementTime = time.Now()
	defer ticker.Stop()
	for {
		select {
		case <-ctx.Done():
			log.Infof("Done channel signaled, quitting. %s", ctx.Err())
			emitStats()
			return
		case <-ticker.C:
			emitStats()
		}
	}
}

func EnablePprof(bindAddress string) {
	go func() {
		var httpAddress string
		if bindAddress[0] == ':' {
			httpAddress = fmt.Sprintf("localhost%s", bindAddress)
		} else {
			httpAddress = bindAddress
		}
		log.Infof("Enabling pprof, bound to %s. \t open http://%s/debug/pprof/", bindAddress, httpAddress)
		log.Error(http.ListenAndServe(bindAddress, nil))
	}()
}

func success() {
	atomic.AddUint64(&successfulRequests, 1)
}

func fail() {
	atomic.AddUint64(&failedRequests, 1)
}

func discard() {
	atomic.AddUint64(&discardedLogLines, 1)
}

func emitStats() {
	now := time.Now()
	timePassed := now.Sub(lastMeasurementTime)
	lastMeasurementTime = now
	success := atomic.LoadUint64(&successfulRequests)
	failed := atomic.LoadUint64(&failedRequests)
	disacarded := atomic.LoadUint64(&discardedLogLines)
	deltaSent := success + failed - lastRequestCount
	lastRequestCount = success + failed
	sendRate := uint64(math.Round((float64(deltaSent) / float64(timePassed)) * float64(time.Second)))
	passedLatencySinceEpocMs := atomic.SwapUint64(&latencySinceEpochMs, 0)
	var avgLatencyMs int64
	if deltaSent == 0 {
		avgLatencyMs = -1
	} else {
		avgLatencyMs = int64(passedLatencySinceEpocMs / deltaSent)
	}
	log.Infof("\t\tSTATS: success: %d, failed: %d, discarded: %d. Total lines: %d. Total sent: %d. \t Throughput: %d/sec \t Latency: %dms",
		success, failed, disacarded, success+failed+disacarded, success+failed, sendRate, avgLatencyMs)
}
