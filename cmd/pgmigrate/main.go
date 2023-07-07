package main

import (
	"fmt"
	"os"

	"github.com/fatih/color"

	"github.com/peterldowns/pgmigrate/cmd/pgmigrate/root"
)

func main() {
	defer func() {
		switch t := recover().(type) {
		case error:
			onError(fmt.Errorf("panic: %w", t))
		case string:
			onError(fmt.Errorf("panic: %s", t))
		default:
			if t != nil {
				onError(fmt.Errorf("panic: %+v", t))
			}
		}
		os.Exit(0)
	}()
	if err := root.Command.Execute(); err != nil {
		onError(err)
	}
}

func onError(err error) {
	msg := fmt.Sprintf("error: %s", err)
	_, _ = fmt.Fprintln(os.Stderr, color.New(color.FgRed, color.Italic).Sprintf(msg))
	os.Exit(1) //nolint:revive // intentional error handling
}
