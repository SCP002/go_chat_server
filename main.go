package main

import (
	"fmt"
	"net/http"
	"os"

	"go_chat_server/chat"
	"go_chat_server/cli"
	"go_chat_server/config"
	"go_chat_server/logger"
	fileUtil "go_chat_server/util/file"
	stdinUtil "go_chat_server/util/stdin"

	goFlags "github.com/jessevdk/go-flags"
	"github.com/sirupsen/logrus"
)

func main() {
	log := logger.New(logrus.FatalLevel, os.Stderr)

	flags, err := cli.Parse()
	if flags.Version {
		fmt.Println("v0.1.0")
		os.Exit(0)
	}
	if cli.IsErrOfType(err, goFlags.ErrHelp) {
		// Help message will be prined by go-flags
		os.Exit(0)
	}
	if err != nil {
		log.Fatal(err)
	}

	log.SetLevel(flags.LogLevel)

	cfg, err := config.Read()
	if err != nil {
		log.Debug(err)
	}

	if cfg.ListenAddress == "" {
		cfg.ListenAddress = stdinUtil.AskListenAddress(log)
	}

	err = config.Write(cfg)
	if err != nil {
		log.Error(err)
	}

	log.Infof("Starting server at %v", cfg.ListenAddress)

	chatHandler := chat.NewHandler(log)
	http.HandleFunc("/chat", chatHandler.Chat)

	certFile := "chat.crt"
	keyFile := "chat.key"
	if fileUtil.Exist(certFile) && fileUtil.Exist(keyFile) {
		log.Fatal(http.ListenAndServeTLS(cfg.ListenAddress, certFile, keyFile, nil))
	}
	log.Fatal(http.ListenAndServe(cfg.ListenAddress, nil))
}
