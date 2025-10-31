package main

import (
	"context"
	"echodb/internal/app"
	conf "echodb/internal/config"
	"echodb/pkg/logging"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
)

var version = "1.0.0"
var showVersion bool

func init() {
	flag.BoolVar(&showVersion, "version", false, "Print version info")
}

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	sign := make(chan os.Signal, 1)
	signal.Notify(sign, syscall.SIGINT, syscall.SIGTERM)

	go func() {
		<-sign
		cancel()
	}()

	configPath := flag.String("config", "./config.yaml", "The path to the configuration file")
	dbName := flag.String("db", "", "Name of the backup database")
	all := flag.Bool("all", false, "Backup of all databases from the configuration")
	fileLog := flag.String("file-log", "echodb.log", "Log files from the configuration")

	flag.Parse()

	if showVersion {
		fmt.Printf("echodb version %s\n", version)
		return
	}

	env := app.Env{
		ConfigFile: *configPath,
		DbName:     *dbName,
		All:        *all,
		FileLog:    *fileLog,
	}

	config, err := conf.Load(*configPath)
	if err != nil {
		fmt.Printf("configuration loading error : %v", err)
		os.Exit(1)
	}
	logger := runLog(&env, *config.Settings.Logging)

	defer func(logger *logging.Logs) {
		_ = logger.Close()
	}(logger)

	ctx = logging.ContextWithLogger(ctx, logger.Logger)

	logging.L(ctx).Info("Configuration loaded successfully")

	a := app.NewApp(ctx, config, &env)

	logging.L(ctx).Info("Starting the application...")

	if err := a.MustRun(); err != nil {
		logging.L(ctx).Error("Application failed to run", logging.ErrAttr(err))
		fmt.Printf("application failed to run: %v", err)
		os.Exit(1)
	}

	logging.L(ctx).Info("Finished dump...")
	os.Exit(0)
}

func runLog(env *app.Env, isLogging bool) *logging.Logs {
	var opts = []logging.LoggerOption{
		logging.WithFile(env.FileLog),
		logging.WithEnabled(isLogging),
		logging.WithAddSource(false),
		logging.WithLevel("debug"),
		logging.WithIsJSON(true),
	}

	return logging.NewLogger(opts...)
}
