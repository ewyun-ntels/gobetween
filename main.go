/**
 * main.go - entry point
 * @author Yaroslav Pogrebnyak <yyyaroslav@gmail.com>
 */
package main

import (
	"log"
	"math/rand"
	"os"
	"os/signal"
	"runtime"
	"syscall"
	"time"

	"github.com/yyyar/gobetween/cmd"
	"github.com/yyyar/gobetween/config"
	"github.com/yyyar/gobetween/info"
	"github.com/yyyar/gobetween/logging"
	"github.com/yyyar/gobetween/manager"
	"github.com/yyyar/gobetween/metrics"
	"github.com/yyyar/gobetween/multiprocess"
	"github.com/yyyar/gobetween/utils/codec"
)

/**
 * version,revision,branch should be set while build using ldflags (see Makefile)
 */
var (
	version  string
	revision string
	branch   string
)

/**
 * Initialize package
 */
func init() {

	// Set GOMAXPROCS if not set
	if os.Getenv("GOMAXPROCS") == "" {
		runtime.GOMAXPROCS(runtime.NumCPU())
	}

	// Init random seed
	rand.Seed(time.Now().UnixNano())

	// Save info
	info.Version = version
	info.Revision = revision
	info.Branch = branch
	info.StartTime = time.Now()

}

/**
 * Entry point
 */
func main() {
	// The bootstrap applies CPU affinity and execs a fresh worker before any
	// configuration parsing or pidfile handling takes place.
	if multiprocess.IsBootstrap() {
		if err := multiprocess.ExecWorkerWithAffinity(); err != nil {
			log.Fatal("Failed to bootstrap worker: ", err)
		}
		return
	}

	log.Printf("gobetween v%s", version)

	env := os.Getenv("GOBETWEEN")
	if env != "" && len(os.Args) > 1 {
		log.Fatal("Passed GOBETWEEN env var and command-line arguments: only one allowed")
	}

	// Try parse env var to args
	if env != "" {
		a := []string{}
		if err := codec.Decode(env, &a, "json"); err != nil {
			log.Fatal("Error converting env var to parameters: ", err, " ", env)
		}
		os.Args = append([]string{""}, a...)
		log.Println("Using parameters from env var: ", os.Args)
	}

	// Process flags and start
	cmd.Execute(func(cfg *config.Config) {

		// Configure logging
		logging.Configure(cfg.Logging.Output, cfg.Logging.Level, cfg.Logging.Format)

		if multiprocess.IsWorker() {
			if err := multiprocess.RunWorker(*cfg); err != nil {
				log.Fatal("Worker failed: ", err)
			}
			return
		}

		if cfg.Runtime.WorkerProcesses > 0 {
			if err := multiprocess.RunSupervisor(*cfg); err != nil {
				log.Fatal("Supervisor failed: ", err)
			}
			return
		}

		// Legacy single-process mode remains available when [runtime] is absent.
		metrics.Start(cfg.Metrics)
		manager.Initialize(*cfg)

		signals := make(chan os.Signal, 1)
		signal.Notify(signals, syscall.SIGINT, syscall.SIGTERM)
		<-signals
		signal.Stop(signals)
		manager.StopAll()
	})
}
