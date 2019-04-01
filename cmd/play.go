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
	logFile    *string
	rate       *int64
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
		err = play.PlayLogFile(ctx, target, *logFile, ratelimiter.Limit(*rate))
		if err != nil {
			log.Errorf("Error %+v", err)
		}
	},
}

func init() {
	rootCmd.AddCommand(playCmd)

	targetHost = addRequiredStringFlag("target-host", "", "Target host to which paly traffic to, scheme://host:port (e.g. http://localhost:1235)")
	logFile = addRequiredStringFlag("log-file", "", "Location of the log file")
	rate = playCmd.Flags().Int64("rate", 0, "The rate at which request are made (requests per second). If <= 0 (or not provided) then rate is not limited")
}

func addRequiredStringFlag(name, value, usage string) *string {
	ref := playCmd.Flags().String(name, value, usage)
	err := playCmd.MarkFlagRequired(name)
	if err != nil {
		panic(err)
	}
	return ref
}
