package cmd

import (
	"context"
	"net/url"

	log "github.com/sirupsen/logrus"
	"github.com/spf13/cobra"
	ratelimiter "golang.org/x/time/rate"

	"gitlab.appsflyer.com/rantav/elb-log-replay/play"
)

// Flags
var (
	targetHost *string
	logFiles   *string
	rate       *int64
	numSenders *uint
)

// playCmd represents the play command
var playCmd = &cobra.Command{
	Use:   "play",
	Short: "Play the an ELB log",
	Run: func(cmd *cobra.Command, args []string) {
		target, err := url.Parse(*targetHost)
		if err != nil {
			log.Fatalf("Cannot parse target URL %s. %+v", *targetHost, err)
		}
		ctx := context.Background()
		err = play.PlayLogFiles(ctx, target, *logFiles, ratelimiter.Limit(*rate), *numSenders)
		if err != nil {
			log.Errorf("Error %+v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(playCmd)

	targetHost = addRequiredStringFlag("target-host", "", "Target host to which paly traffic to, scheme://host:port (e.g. http://localhost:1235)")
	logFiles = addRequiredStringFlag("log-files", "", "Location of the log files. We look for all files in this path ending with *.txt")
	rate = playCmd.Flags().Int64("rate", 0, "The rate at which request are made (requests per second). If <= 0 (or not provided) then rate is not limited")
	numSenders = playCmd.Flags().Uint("num-senders", 32, "The number of HTTP executors (senders). This is the number of parallel HTTP clients that send HTTP requests")
}

func addRequiredStringFlag(name, value, usage string) *string {
	ref := playCmd.Flags().String(name, value, usage)
	err := playCmd.MarkFlagRequired(name)
	if err != nil {
		panic(err)
	}
	return ref
}
