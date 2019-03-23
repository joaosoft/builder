package builder

import (
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"os/exec"

	"io"
	"io/ioutil"

	"github.com/joaosoft/logger"
	"github.com/joaosoft/manager"
	"github.com/joaosoft/watcher"
)

type Builder struct {
	config        *BuilderConfig
	event         chan *watcher.Event
	isLogExternal bool
	pm            *manager.Manager
	mux           sync.Mutex
	logger        logger.ILogger
	reloadTime    int64
	quit          chan int
	started       bool
}

func NewBuilder(options ...BuilderOption) *Builder {
	config, simpleConfig, err := NewConfig()
	pm := manager.NewManager(manager.WithRunInBackground(true))
	event := make(chan *watcher.Event)

	service := &Builder{
		event:      event,
		reloadTime: 1,
		pm:         pm,
		logger:     logger.NewLogDefault("builder", logger.WarnLevel),
		quit:       make(chan int),
		config:     config.Builder,
	}

	w := watcher.NewWatcher(watcher.WithLogger(service.logger), watcher.WithManager(pm), watcher.WithEventChannel(event))
	pm.AddProcess("watcher", w)

	if err != nil {
		service.logger.Error(err.Error())
	} else if config.Builder != nil {
		service.pm.AddConfig("config_app", simpleConfig)
		level, _ := logger.ParseLevel(config.Builder.Log.Level)
		service.logger.Debugf("setting log level to %s", level)
		service.logger.Reconfigure(logger.WithLevel(level))
	}

	if service.isLogExternal {
		service.pm.Reconfigure(manager.WithLogger(service.logger))
	}

	service.Reconfigure(options...)

	return service
}

// execute ...
func (b *Builder) execute() error {
	b.logger.Debug("executing builder")

	termChan := make(chan os.Signal, 1)
	signal.Notify(termChan, syscall.SIGINT, syscall.SIGTERM, syscall.SIGUSR1)

	go func() {
		for {
			select {
			case <-termChan:
				b.logger.Info("received term signal")
				return
			case <-b.quit:
				b.logger.Info("received shutdown signal")
				return
			case <-time.After(time.Duration(b.reloadTime) * time.Second):
				b.logger.Info("watching changes...")

				ev := <-b.event
				if ev.Operation == watcher.OperationChanges {
					b.logger.Infof("%s file %s", ev.Operation, ev.File)
					b.build()
					b.start()
				}
			}
		}
	}()

	return nil
}

// build ...
func (b *Builder) build() error {
	b.logger.Info("executing build")
	cmd := exec.Command("go", "build", "-i", "-o", b.config.Destination, b.config.Source)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return b.logger.Errorf("error getting stderr pipe %s", err).ToError()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return b.logger.Errorf("error getting stdout pipe %s", err).ToError()
	}

	if err = cmd.Start(); err != nil {
		return b.logger.Errorf("error executing build command %s", err).ToError()
	}

	io.Copy(os.Stdout, stdout)
	errBuf, _ := ioutil.ReadAll(stderr)

	if err = cmd.Wait(); err != nil {
		return b.logger.Errorf("error executing build %s", string(errBuf)).ToError()
	}
	b.logger.Info("build completed")

	return nil
}

// start ...
func (b *Builder) start() error {
	b.logger.Info("executing start")
	cmd := exec.Command("./" + b.config.Destination)

	stderr, err := cmd.StderrPipe()
	if err != nil {
		return b.logger.Errorf("error getting stderr pipe %s", err).ToError()
	}

	stdout, err := cmd.StdoutPipe()
	if err != nil {
		return b.logger.Errorf("error getting stdout pipe %s", err).ToError()
	}

	if err = cmd.Start(); err != nil {
		return b.logger.Errorf("error executing restart command %s", err).ToError()
	}

	io.Copy(os.Stdout, stdout)
	errBuf, _ := ioutil.ReadAll(stderr)

	if err = cmd.Wait(); err != nil {
		return b.logger.Errorf("error executing restart %s", string(errBuf)).ToError()
	}
	b.logger.Info("start completed")

	return nil
}

// Start ...
func (b *Builder) Start(waitGroup ...*sync.WaitGroup) error {
	var wg *sync.WaitGroup

	if len(waitGroup) == 0 {
		wg = &sync.WaitGroup{}
		wg.Add(1)
	} else {
		wg = waitGroup[0]
	}

	defer wg.Done()

	if err := b.pm.Start(); err != nil {
		return err
	}

	b.started = true
	if err := b.execute(); err != nil {
		return err
	}

	return nil
}

// Started ...
func (b *Builder) Started() bool {
	return b.started
}

// Stop ...
func (b *Builder) Stop(waitGroup ...*sync.WaitGroup) error {
	var wg *sync.WaitGroup

	if len(waitGroup) == 0 {
		wg = &sync.WaitGroup{}
		wg.Add(1)
	} else {
		wg = waitGroup[0]
	}

	defer wg.Done()

	b.started = false
	if err := b.pm.Stop(); err != nil {
		return err
	}

	return nil
}
