package core

import (
	"bytes"
	"encoding/json"
	"go.uber.org/zap/zapcore"
	"net/http"
	"os"

	"go.uber.org/zap"
)

func NewLogger() (*zap.SugaredLogger, error) {
	var logger *zap.Logger
	var err error

	if os.Getenv("ENVIRONMENT") == "production" {
		logger, err = zap.NewProduction()
		if err != nil {
			return nil, err
		}
	} else {
		logger, err = zap.NewDevelopment()
		if err != nil {
			return nil, err
		}
	}

	betterStackLogsHook := func(entry zapcore.Entry) error {
		token := os.Getenv("BETTERSTACK_LOGS_API_KEY")
		if token != "" {
			logEntry := struct {
				Message  string `json:"message"`
				Level    string `json:"level"`
				Function string `json:"function"`
				File     string `json:"file"`
			}{
				Message:  entry.Message,
				Level:    entry.Level.String(),
				Function: entry.Caller.Function,
				File:     entry.Caller.File,
			}

			logEntryJSON, err := json.Marshal(logEntry)
			if err != nil {
				return err
			}

			req, err := http.NewRequest("POST", "https://in.logs.betterstack.com/", bytes.NewBuffer(logEntryJSON))
			if err != nil {
				return err
			}

			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer "+token)

			client := http.Client{}
			resp, err := client.Do(req)
			if err != nil {
				return err
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusAccepted {
				return err
			}
		}

		return nil
	}

	logger = logger.WithOptions(zap.Hooks(betterStackLogsHook))

	return logger.Sugar(), nil
}
