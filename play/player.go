package play

import (
	"bufio"
	"context"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	ratelimiter "golang.org/x/time/rate"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// LineLog is the struct to analyce and store a line
type logLine struct {
	url        string
	method     string
	userAgent  string
	statusCode int
}

// PlayLogFile plays an ELB log file at a given rate (per second)
// path is a path to all log files. We would look for all log files ending with txt, e.g. path/*.txt
func PlayLogFiles(ctx context.Context, target *url.URL, path string, rate ratelimiter.Limit) error {
	files := findFiles(path)
	go monitor(ctx, monitoringFrequency)
	defer emitStats()
	rateLimiter := createRateLimiter(rate)
	for _, file := range files {
		err := replayLogFile(ctx, target, file, rateLimiter)
		if err != nil {
			log.Errorf("Error playing file %s: %+v", file, err)
		}
	}
	return nil
}

// creates a rate limiter. If the rate is 0 or less, returns nill (meaning no limit)
func createRateLimiter(rate ratelimiter.Limit) *ratelimiter.Limiter {
	var rateLimiter *ratelimiter.Limiter
	if rate > 0 {
		// Allow burst of 1/10
		burst := int(rate / 10)
		if burst == 0 {
			burst = 1
		}
		rateLimiter = ratelimiter.NewLimiter(rate, burst)
	}
	return rateLimiter
}

func findFiles(path string) []string {
	path = fmt.Sprintf("%s/*.txt", path)
	matches, err := filepath.Glob(path)
	if err != nil {
		panic(err)
	}
	log.Infof("Found %d log files", len(matches))
	return matches
}

// Replays a single log file
func replayLogFile(ctx context.Context, target *url.URL, filePath string, rateLimiter *ratelimiter.Limiter) error {
	log.Infof("opening log file %s", filePath)

	f, err := os.Open(filePath)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()

	rdr := bufio.NewReader(f)

	for {
		b, err := rdr.ReadBytes('\n')
		if err != nil && err != io.EOF {
			return errors.WithStack(err)
		}

		if len(b) > 0 {
			reqLine := strings.TrimSpace(string(b))
			log.Debugf("replaying line: %s", reqLine)
			err := send(ctx, target, reqLine, rateLimiter)
			if err != nil {
				log.Errorf("%s", err)
				continue
			}
		}

		if err == io.EOF {
			break
		}
	}
	log.Info("Done with file")
	return nil
}

func send(ctx context.Context, target *url.URL, line string, limiter *ratelimiter.Limiter) error {
	if limiter != nil {
		err := limiter.Wait(ctx)
		if err != nil {
			return errors.WithStack(err)
		}
	}

	logLine, err := parse(line)
	if err != nil {
		return errors.WithStack(err)
	}
	if logLine.statusCode < 200 || logLine.statusCode >= 300 {
		// discard
		log.Debugf("Discarding non 2xx line %v", logLine)
		return nil
	}
	go func() {
		err := sendRequest(ctx, target, logLine)
		if err != nil {
			log.Errorf("Error sending %+v", err)
		}
	}()

	return nil
}
