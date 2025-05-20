package logger

import (
	"fmt"
	"os"

	"github.com/fatih/color"
)

type Logger struct {
	Verbose bool
	Debug   bool
}

func (l Logger) Infof(msg string, args ...any) {
	if l.Verbose {
		fmt.Fprintf(os.Stdout, color.GreenString("[info] ")+msg+"\n", args...)
	}
}

func (l Logger) Debugf(msg string, args ...any) {
	if l.Debug {
		fmt.Fprintf(os.Stdout, color.CyanString("[debug] ")+msg+"\n", args...)
	}
}

func (l Logger) Warnf(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, color.YellowString("[warn] ")+msg+"\n", args...)
}

func (l Logger) Errorf(msg string, args ...any) {
	fmt.Fprintf(os.Stderr, color.RedString("[error] ")+msg+"\n", args...)
}
