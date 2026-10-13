package telegrambot

import (
	"fmt"
	"net"
	"net/http"
	"strings"
	"time"

	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"go.uber.org/zap"
	"go.uber.org/zap/zapcore"
)

func New(log *zap.Logger, token string) (*tgbotapi.BotAPI, error) {
	if err := tgbotapi.SetLogger(&BotLogger{log}); err != nil {
		return nil, err
	}
	// Таймауты клиента должны быть больше long-poll таймаута getUpdates
	// (см. u.Timeout в startBot), иначе клиент будет обрывать сам long-poll
	// запрос раньше, чем Telegram успеет на него ответить.
	api, err := tgbotapi.NewBotAPIWithClient(token, tgbotapi.APIEndpoint, &http.Client{
		Timeout: 40 * time.Second,
		Transport: &http.Transport{
			DialContext: (&net.Dialer{
				Timeout:   5 * time.Second,
				KeepAlive: 30 * time.Second,
			}).DialContext,
			TLSHandshakeTimeout:   time.Second * 3,
			ResponseHeaderTimeout: time.Second * 35,
			ExpectContinueTimeout: 1 * time.Second,
			IdleConnTimeout:       60 * time.Second,
		},
	})
	if err != nil {
		return nil, err
	}
	if zapcore.Level(log.Level()) == zapcore.DebugLevel {
		api.Debug = true
	}
	return api, nil
}

type BotLogger struct {
	*zap.Logger
}

func (b *BotLogger) Println(v ...any) {
	switch len(v) {
	case 0:
		return
	case 1:
		mes, ok := v[0].(string)
		if ok {
			if strings.Contains(strings.ToLower(mes), "failed") {
				b.Error(mes)
			} else {
				b.Debug(mes)
			}
		} else {
			err, okErr := v[0].(*tgbotapi.Error)
			if okErr {
				b.Error("telegram api error",
					zap.Int("code", err.Code),
					zap.String("message", err.Message),
					zap.Int("retry after seconds", err.ResponseParameters.RetryAfter),					
				)
			} else {
				b.Warn("something went wrong",
					zap.Any("params", v[0]),
				)
			}
		}

	default:
		fields := []string{fmt.Sprintf("%s", v[0])}
		for i := 1; i < len(v); i++ {
			fields = append(fields, fmt.Sprintf("%s", v[i]))
		}
		b.Debug(strings.Join(fields, ":"))
	}
}

func (b *BotLogger) Printf(format string, v ...any) {
	b.Debug(fmt.Sprintf(format, v...))
}

// func NewUpdate(offset int) tgbotapi.UpdateConfig {
// 	return tgbotapi.NewUpdate(offset)
// }
