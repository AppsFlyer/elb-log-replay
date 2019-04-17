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
func PlayLogFiles(
	ctx context.Context,
	target *url.URL,
	path string,
	rate ratelimiter.Limit,
	numSenders uint,
) error {
	files := findFiles(path)
	go monitor(ctx, monitoringFrequency)
	defer emitStats()
	rateLimiter := createRateLimiter(rate)
	lines := generateLogLines(ctx, files, rateLimiter, 2*numSenders)
	var dones []<-chan struct{}
	for i := uint(0); i <= numSenders; i++ {
		done := sendLines(ctx, target, lines)
		dones = append(dones, done)
	}
	waitForAll(dones)
	return nil
}

func generateLogLines(
	ctx context.Context,
	files []string,
	rateLimiter *ratelimiter.Limiter,
	bufferSize uint,
) <-chan string {
	lines := make(chan string, bufferSize)
	go func() {
		for _, file := range files {
			err := replayLogFile(ctx, file, rateLimiter, lines)
			if err != nil {
				log.Errorf("Error playing file %s: %+v", file, err)
			}
		}
		close(lines)
	}()
	return lines
}

func sendLines(
	ctx context.Context,
	target *url.URL,
	lines <-chan string,
) <-chan struct{} {
	done := make(chan struct{})
	go func() {
		defer close(done)
		for line := range lines {
			logLine, err := parse(line)
			if err != nil {
				discard()
				log.Debugf("Error parsing line %s. \t %s", line, err)
				continue
			}
			if logLine.statusCode < 200 || logLine.statusCode >= 400 {
				discard()
				log.Debugf("Discarding non 2xx or 3xx line %v", logLine)
				continue
			}
			err = sendRequest(ctx, target, logLine)
			if err != nil {
				log.Errorf("Error sending %+v", err)
			}
		}
	}()
	return done
}

func waitForAll(cs []<-chan struct{}) {
	for _, c := range cs {
		<-c
	}
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
func replayLogFile(
	ctx context.Context,
	filePath string,
	limiter *ratelimiter.Limiter,
	lines chan<- string,
) error {
	log.Infof("Playing log file %s", filePath)

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
			line := strings.TrimSpace(string(b))
			err = limiter.Wait(ctx)
			if err != nil {
				log.Errorf("Error waiting for rate limiter. %s", err)
			}
			lines <- line
		}

		if err == io.EOF {
			break
		}
	}
	log.Infof("Done with file %s", filePath)
	return nil
}
