/*
Copyright © 2022
Author Bhakiyaraj Kalimuthu
Email bhakiya.kalimuthu@gmail.com
*/
package cmd

import (
	"context"
	"fmt"
	"github.com/mattn/go-colorable"
	"log"
	"net/url"
	"os"
	"os/signal"
	"sigin/internal"
	"sigin/store"
	"sync"
	"syscall"
	"time"

	"github.com/spf13/cobra"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

const (
	workerPoolSize = 10    // default worker pool size
	env            = "dev" // development or production env
)

// rootCmd represents the base command when called without any subcommands
var (
	rootCmd = &cobra.Command{
		Use:   "sigin",
		Short: "method signature  inserter",
		Long:  `sigin can get eth method signature from configured URL and insert into db URL`,
		Run:   runRootCmd,
	}
	rootArgs struct {
		serverAddress string // url where method signature available
		dbAddress     string // db address
	}
)

// Execute adds all child commands to the root command and sets flags appropriately.
// This is called by main.main(). It only needs to happen once to the rootCmd.
func Execute() {
	err := rootCmd.Execute()
	if err != nil {
		os.Exit(1)
	}
}

func init() {
	root := rootCmd.Flags()
	root.StringVarP(&rootArgs.serverAddress, "server-address", "s", "https://www.4byte.directory/api/v1/signatures", "server address where method signature available")
	root.StringVarP(&rootArgs.dbAddress, "db-address", "d", "localhost", "database address where method signature needs to be stored")
	//cobra.MarkFlagRequired(root, "server-address")
	//cobra.MarkFlagRequired(root, "db-address")
}

func runRootCmd(cmd *cobra.Command, args []string) {
	// logger setup
	l := loggerSetup()

	// init clock
	clock := internal.NewClock()
	defer func() {
		l.Info("Time taken to complete", zap.Duration("time_taken", <-clock.Since()))
	}()

	// validate url and fail early
	if !isValidURL(rootArgs.serverAddress) {
		cmd.Help()
		os.Exit(1)
	}

	// producer channel
	pChan := make(chan *internal.Response, 1)
	// consumer channel
	cChan := make(chan *internal.Response, workerPoolSize)

	db := store.NewPostgres(rootArgs.dbAddress)
	// create http client
	httpClient := internal.NewHttpClient(l, db)

	// create siginer
	sigin := internal.NewSigin(l, httpClient, db, time.Millisecond, pChan, cChan, rootArgs.serverAddress)

	// setup cancellation context and wait group
	// root background with cancellation support
	ctx, cancel := context.WithCancel(context.Background())
	wg := new(sync.WaitGroup)

	// start sigin and pass the cancellation ctx
	go sigin.Start(ctx)

	// start workers and add worker pool
	wg.Add(workerPoolSize)
	for i := 1; i <= workerPoolSize; i++ {
		go sigin.Process(wg, i)
	}

	doneCh := make(chan os.Signal, 1)

	// user input
	go func(doneCh chan os.Signal) {
		sigin.CreateJobs()
		<-time.Tick(time.Second * 3) // wait for all the workers to finish up
		// exit the program
		doneCh <- syscall.SIGQUIT
	}(doneCh)

	// handle manual interruption
	signal.Notify(doneCh, syscall.SIGINT, syscall.SIGTERM)

	switch <-doneCh { // blocks here until interrupted
	case syscall.SIGINT, syscall.SIGTERM:
		l.Warn("CTRL-C received.Terminating......")
	default:
		l.Warn("file read is completed,exiting......")
	}
	signal.Stop(doneCh)

	// handle shut down
	cancel() // cancel context
	// even if cancellation received, current running job will be not be interrupted until it completes
	wg.Wait() // wait for the workers to be completed
	l.Warn("All jobs are done, shutting down")

}

// loggerSetup setup zap logger
func loggerSetup() *zap.Logger {
	if env == "prod" {
		logger, err := zap.NewProduction()
		if err != nil {
			log.Fatalf("failed to create zap logger : %v", err)
		}
		logger.Info("logger setup done")
		return logger
	}

	// setup dev logger to show different colors
	cfg := zap.NewDevelopmentEncoderConfig()
	cfg.EncodeLevel = zapcore.CapitalColorLevelEncoder
	log := zap.New(zapcore.NewCore(
		zapcore.NewConsoleEncoder(cfg),
		zapcore.AddSync(colorable.NewColorableStdout()),
		zapcore.DebugLevel,
	))
	log.Info("logger setup done")
	return log
}

func isValidURL(URL string) bool {
	if URL == "" {
		fmt.Println("Error: url field is empty")
		return false
	}

	// parse url if valid
	_, err := url.ParseRequestURI(URL)
	if err != nil {
		fmt.Printf("Error: invalid url %v", err)
		return false
	}
	return true
}
