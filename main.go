package main

import (
	"context"
	"flag"
	"fmt"
	"log"
	"os"
	"os/signal"
	"strings"
	"syscall"

	reloader "memphis-config-reloader/reloader"
)

var (
	BuildTime = "build-time-not-set"
	GitInfo   = "gitinfo-not-set"
	Version   = "version-not-set"
)

// StringSet is a wrapper for []string to allow using it with the flags package.
type StringSet []string

func (s *StringSet) String() string {
	return strings.Join([]string(*s), ", ")
}

// Set appends the value provided to the list of strings.
func (s *StringSet) Set(val string) error {
	*s = append(*s, val)
	return nil
}

func main() {
	fs := flag.NewFlagSet("memphis-config-reloader", flag.ExitOnError)
	flag.Usage = func() {
		fmt.Fprintf(os.Stderr, "Usage: memphis-config-reloader [options...]\n\n")
		fs.PrintDefaults()
		fmt.Fprintf(os.Stderr, "\n")
	}

	// Help and version
	var (
		showHelp     bool
		showVersion  bool
		fileSet      StringSet
		customSignal int
	)

	nconfig := &reloader.Config{}
	fs.BoolVar(&showHelp, "h", false, "Show help")
	fs.BoolVar(&showHelp, "help", false, "Show help")
	fs.BoolVar(&showVersion, "v", false, "Show version")
	fs.BoolVar(&showVersion, "version", false, "Show version")

	fs.StringVar(&nconfig.PidFile, "P", "/var/run/nats/gnatsd.pid", "Memphis Pid File")
	fs.StringVar(&nconfig.PidFile, "pid", "/var/run/nats/gnatsd.pid", "Memphis Pid File")
	fs.Var(&fileSet, "c", "Memphis Config File (may be repeated to specify more than one)")
	fs.Var(&fileSet, "config", "Memphis Config File (may be repeated to specify more than one)")
	fs.IntVar(&nconfig.MaxRetries, "max-retries", 30, "Max attempts to trigger reload")
	fs.IntVar(&nconfig.RetryWaitSecs, "retry-wait-secs", 4, "Time to back off when reloading fails before retrying")
	fs.IntVar(&customSignal, "signal", 1, "Signal to send to the Memphis process (default SIGHUP 1)")

	fs.Parse(os.Args[1:])

	nconfig.ConfigFiles = fileSet
	if len(fileSet) == 0 {
		nconfig.ConfigFiles = []string{"/etc/nats-config/gnatsd.conf"}
	}
	nconfig.Signal = syscall.Signal(customSignal)

	switch {
	case showHelp:
		flag.Usage()
		os.Exit(0)
	case showVersion:
		fmt.Fprintf(os.Stderr, "Memphis Config Reloader v%s (%s, %s)\n", Version, GitInfo, BuildTime)
		os.Exit(0)
	}
	r, err := reloader.NewReloader(nconfig)
	if err != nil {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err)
		os.Exit(1)
	}

	// Signal handling.
	go func() {
		c := make(chan os.Signal, 1)
		signal.Notify(c, syscall.SIGINT, syscall.SIGTERM)

		for sig := range c {
			log.Printf("Trapped \"%v\" signal\n", sig)
			switch sig {
			case syscall.SIGINT:
				log.Println("Exiting...")
				os.Exit(0)
				return
			case syscall.SIGTERM:
				r.Stop()
				return
			}
		}
	}()

	log.Printf("Starting Memphis Config Reloader v%s\n", Version)
	err = r.Run(context.Background())
	if err != nil && err != context.Canceled {
		fmt.Fprintf(os.Stderr, "Error: %s\n", err.Error())
		os.Exit(1)
	}
}
