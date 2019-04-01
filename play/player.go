package play

import (
	"bufio"
	"context"
	"io"
	"net/url"
	"os"
	"regexp"
	"strings"

	ratelimiter "golang.org/x/time/rate"

	"github.com/pkg/errors"
	log "github.com/sirupsen/logrus"
)

// LineLog is the struct to analyce and store a line
type logLine struct {
	url       string
	method    string
	userAgent string
}

var (
	reClasic      *regexp.Regexp
	matcherClasic []string
)

func init() {
	reClasic = regexp.MustCompile(`(?P<date>[^Z]+Z) (?P<elb>[^\s]+) (?P<ipclient>[^:]+?):[0-9]+ (?P<ipnode>[^:]+?):[0-9]+ (?P<reqtime>[0-9\.]+) (?P<backtime>[0-9\.]+) (?P<restime>[0-9\.]+) (?P<elbcode>[0-9]{3}) (?P<backcode>[0-9]{3}) (?P<lenght1>[0-9]+) (?P<lenght2>[0-9]+) "(?P<Method>[A-Z]+) (?P<URL>[^"]+) HTTP/[0-9\.]+" "(?P<useragent>[^"]+)" .*`)
	matcherClasic = reClasic.SubexpNames()
}

// PlayLogFile plays an ELB log file at a given rate (per second)
func PlayLogFile(ctx context.Context, target *url.URL, path string, rate ratelimiter.Limit) error {
	log.Infof("opening log file %s", path)

	f, err := os.Open(path)
	if err != nil {
		return errors.WithStack(err)
	}
	defer f.Close()
	return replayLogFile(ctx, target, f, rate)
}

// Replays a log file
func replayLogFile(ctx context.Context, target *url.URL, r io.Reader, rate ratelimiter.Limit) error {
	rdr := bufio.NewReader(r)

	var rateLimiter *ratelimiter.Limiter
	if rate > 0 {
		// Allow burst of 1/10
		burst := int(rate / 10)
		if burst == 0 {
			burst = 1
		}
		rateLimiter = ratelimiter.NewLimiter(rate, burst)
	}

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
	go func() {
		err := sendRequest(ctx, target, logLine)
		if err != nil {
			log.Errorf("Error sending %+v", err)
		}
	}()

	return nil
}

// parse the raw record line
func parse(line string) (*logLine, error) {
	matches := reClasic.FindAllStringSubmatch(line, -1)
	if matches == nil {
		return nil, errors.Errorf("Failed to parse the line %s", line)
	}

	return &logLine{
		method:    matches[0][12],
		url:       matches[0][13],
		userAgent: matches[0][14],
	}, nil
}
