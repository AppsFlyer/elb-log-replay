package play

import (
	"context"
	"sync/atomic"
	"time"

	log "github.com/sirupsen/logrus"
)

const monitoringFrequency = 2 * time.Second

var (
	successfulRequests uint64
	failedRequests     uint64
	discardedLogLines  uint64
)

// Loops forever and emits stats every frequency duration
func monitor(ctx context.Context, frequency time.Duration) {
	ticker := time.NewTicker(frequency)
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
	success := atomic.LoadUint64(&successfulRequests)
	failed := atomic.LoadUint64(&failedRequests)
	disacarded := atomic.LoadUint64(&discardedLogLines)
	log.Infof("\tSTATS: success: %d, failed: %d, discarded: %d. Total lines: %d. Total sent: %d",
		success, failed, disacarded, success+failed+disacarded, success+failed)
}
