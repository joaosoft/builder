package main

import (
	"github.com/joaosoft/builder"
	"os"
	"os/signal"
	"syscall"
)

func main() {
	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	build := builder.NewBuilder(builder.WithReloadTime(1))

	if err := build.Start(); err != nil {
		panic(err)
	}

	<-termChan
	if err := build.Stop(); err != nil {
		panic(err)
	}
}
