package tui

import (
	"fmt"
	"io"
	"log"
	"strings"
	"sync"
	"time"

	"github.com/sanity-io/litter"
	"github.com/sirupsen/logrus"
	"gopkg.in/natefinch/lumberjack.v2"
)

// nolint:deadcode,unused
func debug(v any, verbose ...bool) {
	if len(verbose) > 0 && verbose[0] {
		NewLogger().Println(litter.Sdump(v))
		return
	}
	NewLogger().Println(v)
}

// NewLogger returns a new well configured logger.
func NewLogger() logrus.StdLogger {
	formatter := new(logFormatter)

	log := logrus.New()
	log.SetOutput(io.Discard) // stdout & stderr to /dev/null
	log.SetFormatter(formatter)
	log.Hooks.Add(&fileHook{
		rotate: &lumberjack.Logger{
			Filename:   "sfc.log",
			MaxSize:    20, // megabytes
			MaxBackups: 2,
			MaxAge:     10, //days
		},
		formatter: formatter,
	})

	return log
}

////////////////////
//                //
// File hook      //
//                //
////////////////////

type fileHook struct {
	sync.Mutex
	rotate    *lumberjack.Logger
	formatter logrus.Formatter
}

// Fire opens the file, writes to the file and closes the file.
// Whichever user is running the function needs write permissions to the file or directory if the file does not yet exist.
func (hook *fileHook) Fire(entry *logrus.Entry) error {
	hook.Lock()
	defer hook.Unlock()

	// use our formatter instead of entry.String()
	msg, err := hook.formatter.Format(entry)
	if err != nil {
		log.Println("failed to generate string for entry:", err)
		return err
	}

	_, err = hook.rotate.Write(msg)
	return err
}

// Levels returns configured log levels.
func (hook *fileHook) Levels() []logrus.Level {
	return logrus.AllLevels
}

////////////////////
//                //
// Log formatter  //
//                //
////////////////////

type logFormatter struct{}

// Format implements Logrus formatter.
func (f *logFormatter) Format(entry *logrus.Entry) ([]byte, error) {
	fields := ""
	if len(entry.Data) > 0 {
		fs := []string{}
		for k, v := range entry.Data {
			fs = append(fs, fmt.Sprintf("%s=%v", k, v))
		}
		fields = fmt.Sprintf(" (%s)", strings.Join(fs, ", "))
	}

	data := fmt.Sprintf("[%s] %+5s: %s%s\n",
		time.Now().Format(time.RFC3339),
		strings.ToUpper(entry.Level.String()),
		entry.Message,
		fields,
	)
	return []byte(data), nil
}
