package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"strings"

	"geeksaga.com/os/straw/internal"
)

const projectName string = "Straw"

var fVersion = flag.Bool("version", false, "display the version and exit")

var (
	version string
	commit  string
	branch  string
)

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

func main() {
	flag.Usage = func() { usageExit(0) }
	flag.Parse()

	optionHelper()

	shortVersion := version
	if shortVersion == "" {
		shortVersion = "unknown"
	}

	if err := internal.SetVersion(shortVersion); err != nil {
		log.Println(projectName + " version already configured to: " + internal.Version())
	}
}
