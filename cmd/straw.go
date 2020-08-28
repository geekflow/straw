package main

import (
	"context"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	"github.com/geekflow/straw/agent"
	"github.com/geekflow/straw/internal"
	"github.com/geekflow/straw/internal/config"
	"github.com/geekflow/straw/internal/logger"

	_ "github.com/geekflow/straw/plugins/inputs/all"
	_ "github.com/geekflow/straw/plugins/outputs/all"

	log "github.com/sirupsen/logrus"
)

const projectName string = "Straw"

var fConfig = flag.String("config", "", "configuration file to load")
var fConfigDirectory = flag.String("config-directory", "", "directory containing additional *.conf files")
var fVersion = flag.Bool("version", false, "display the version and exit")

var fPidFile = flag.String("pidfile", "", "file to write our pid to")

var fInputList = flag.Bool("input-list", false, "print available input plugins.")
var fOutputList = flag.Bool("output-list", false, "print available output plugins.")

var (
	version string
	commit  string
	branch  string
)

var stop chan struct{}

func reloadLoop(stop chan struct{}) {
	reload := make(chan bool, 1)
	reload <- true
	for <-reload {
		reload <- false

		ctx, cancel := context.WithCancel(context.Background())

		signals := make(chan os.Signal)
		signal.Notify(signals, os.Interrupt, syscall.SIGHUP, syscall.SIGTERM, syscall.SIGINT)
		go func() {
			select {
			case sig := <-signals:
				if sig == syscall.SIGHUP {
					log.Printf("I! Reloading %s config", projectName)
					<-reload
					reload <- true
				}
				cancel()
			case <-stop:
				cancel()
			}
		}()

		err := runAgent(ctx)
		if err != nil && err != context.Canceled {
			log.Fatalf("E! [%s] Error running agent: %v", projectName, err)
		}
	}
}

func usageExit(code int) {
	fmt.Print(internal.Usage)
	os.Exit(code)
}

func formatFullVersion() string {
	var parts = []string{projectName}

	if version != "" {
		parts = append(parts, version)
	} else {
		parts = append(parts, "unknown")
	}

	if branch != "" || commit != "" {
		if branch == "" {
			branch = "unknown"
		}
		if commit == "" {
			commit = "unknown"
		}
		git := fmt.Sprintf("(git: %s %s)", branch, commit)
		parts = append(parts, git)
	}

	return strings.Join(parts, " ")
}

func optionHelper() {
	if *fVersion {
		fmt.Println(formatFullVersion())
		os.Exit(0)
	}
}

func runAgent(ctx context.Context) error {
	log.Printf("I! Starting %s %s", projectName, version)

	c := config.NewConfig()
	err := c.LoadConfig(*fConfig)
	if err != nil {
		return err
	}

	if *fConfigDirectory != "" {
		err = c.LoadDirectory(*fConfigDirectory)
		if err != nil {
			return err
		}
	}

	if len(c.Outputs) == 0 {
		return errors.New("Error: no outputs found, did you provide a valid config file?")
	}

	if int64(c.Agent.Interval.Duration) <= 0 {
		return fmt.Errorf("Agent interval must be positive, found %s",
			c.Agent.Interval.Duration)
	}

	if int64(c.Agent.FlushInterval.Duration) <= 0 {
		return fmt.Errorf("Agent flush_interval must be positive; found %s",
			c.Agent.Interval.Duration)
	}

	ag, err := agent.NewAgent(c)
	if err != nil {
		return err
	}

	// Setup logging as configured.
	logConfig := logger.LogConfig{
		//Level:               ag.Config.Agent.Quiet || *fQuiet,
		Target:              ag.Config.Agent.LogTarget,
		File:                ag.Config.Agent.Logfile,
		RotationInterval:    ag.Config.Agent.LogfileRotationInterval,
		RotationMaxSize:     ag.Config.Agent.LogfileRotationMaxSize,
		RotationMaxArchives: ag.Config.Agent.LogfileRotationMaxArchives,
	}

	logger.InitializeLogging(logConfig)

	log.Printf("I! Loaded inputs: %s", strings.Join(c.InputNames(), " "))
	log.Printf("I! Loaded outputs: %s", strings.Join(c.OutputNames(), " "))
	log.Printf("I! Tags enabled: %s", c.ListTags())

	if *fPidFile != "" {
		f, err := os.OpenFile(*fPidFile, os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			log.Printf("E! Unable to create pidfile: %s", err)
		} else {
			fmt.Fprintf(f, "%d\n", os.Getpid())

			f.Close()

			defer func() {
				err := os.Remove(*fPidFile)
				if err != nil {
					log.Printf("E! Unable to remove pidfile: %s", err)
				}
			}()
		}
	}

	return ag.Run(ctx)
}

func main() {
	flag.Usage = func() { usageExit(0) }
	flag.Parse()

	logger.InitializeLogging(logger.LogConfig{
		Level: log.DebugLevel,
		//	File:  strings.ToLower(projectName) + ".log",
	})

	optionHelper()

	shortVersion := version
	if shortVersion == "" {
		shortVersion = "unknown"
	}

	if err := internal.SetVersion(shortVersion); err != nil {
		log.Println(projectName + " version already configured to: " + internal.Version())
	}

	stop = make(chan struct{})
	reloadLoop(stop)
}
