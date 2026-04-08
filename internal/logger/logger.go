package logger

import (
	"log"
	"os"
)

var (
	Info  *log.Logger
	Warn  *log.Logger
	Error *log.Logger
)

func Init() {
	flags := log.Ldate | log.Ltime | log.Lshortfile

	Info  = log.New(os.Stdout, "INFO  ", flags)
	Warn  = log.New(os.Stdout, "WARN  ", flags)
	Error = log.New(os.Stderr, "ERROR ", flags)
}
