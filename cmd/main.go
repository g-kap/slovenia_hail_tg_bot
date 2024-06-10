package main

import (
	"context"
	"database/sql"
	"flag"
	_ "github.com/mattn/go-sqlite3"
	"log/slog"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"tocadanes/pkg/arsoparser"
	"tocadanes/pkg/storage"
	"tocadanes/pkg/tgbot"
)

var token string
var dbPath string

func main() {
	flag.StringVar(&dbPath, "db", "./db", "path to sqlite3 database file")
	flag.StringVar(&token, "token", "", "telegram bot API token")
	flag.Parse()

	ctx, cancel := context.WithCancel(context.Background())
	log := slog.New(slog.NewTextHandler(os.Stdout, &slog.HandlerOptions{
		AddSource: true,
	}))

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		log.ErrorContext(ctx, "error opening db", "err", err)
		return
	}
	defer db.Close()

	s := storage.NewSqlStorage(db)
	err = s.MaybeInit(ctx)
	if err != nil {
		log.ErrorContext(ctx, "can not init DB", "err", err)
		return
	}
	bot, err := tgbot.NewBot(ctx, token, log, s)
	if err != nil {
		log.ErrorContext(ctx, "can not create TG bot", "err", err)
		return
	}
	parser := arsoparser.NewParser(log)
	var wg sync.WaitGroup

	wg.Add(1)
	go parser.Run(ctx, wg.Done)

	wg.Add(1)
	go bot.Run(ctx, wg.Done)

	go func() {
		for range parser.Changes() {
			log.InfoContext(ctx, "got a new event from arso parser")
			prev, last := parser.PrevState(), parser.LastState()
			for k := range arsoparser.MakeDiff(prev, last) {
				bot.TriggerEvent(ctx, tgbot.NewEventHailProbabilityChange(
					prev[k], last[k], k,
				))
			}
		}
	}()
	sigterm := make(chan os.Signal, 1)
	signal.Notify(sigterm, syscall.SIGINT, syscall.SIGTERM)
	select {
	case <-ctx.Done():
		log.WarnContext(ctx, "terminating via context cancel")
	case <-sigterm:
		log.WarnContext(ctx, "terminating via signal")
	}
	cancel()
	log.WarnContext(ctx, "waiting for the waitgroup")
	wg.Wait()

	log.InfoContext(ctx, "process finished")
}
