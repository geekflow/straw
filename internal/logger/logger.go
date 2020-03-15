package logger

import (
	"fmt"
	"geeksaga.com/os/straw/internal"
	log "github.com/sirupsen/logrus"
	"io"
	"os"
)

type LogConfig struct {
	Level               log.Level
	File                string
	Target              string
	RotationInterval    internal.Duration
	RotationMaxSize     internal.Size
	RotationMaxArchives int
}

func InitializeLogging(config LogConfig) {
	if config.File != "" {
		var file, err = os.OpenFile(config.File, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
		if err != nil {
			fmt.Println("Could Not Open Log File : " + err.Error())
		}

		log.SetOutput(file)
	}

	log.SetFormatter(&log.TextFormatter{
		DisableColors: false,
		FullTimestamp: true,
	})

	log.SetLevel(config.Level)
}

type Log struct {
	log            log.Logger
	internalWriter io.Writer
}

func init() {
}
