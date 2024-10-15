package telegram

import (
	"github.com/gc-9/gf/errors"
	tgbotapi "github.com/go-telegram-bot-api/telegram-bot-api/v5"
	"net/http"
	"net/url"
	"sync"
	"time"
)

type BotConfig struct {
	Proxy  string `yaml:"proxy"`
	Token  string `yaml:"token"`
	ChatId int64  `yaml:"chatId"`
}

func NewBot(config *BotConfig) *Bot {
	bot := &Bot{
		config: config,
	}
	return bot
}

type Bot struct {
	config    *BotConfig
	botApi    *tgbotapi.BotAPI
	lastErr   error
	lastErrAt time.Time
	sync.Mutex
}

func (t *Bot) client() (*tgbotapi.BotAPI, error) {
	client := &http.Client{Timeout: time.Second * 5}
	if t.config.Proxy != "" {
		proxyURL, err := url.Parse(t.config.Proxy)
		if err != nil {
			return nil, err
		}
		client.Transport = &http.Transport{
			Proxy: http.ProxyURL(proxyURL),
		}
	}
	// socks5 := "socks5://127.0.0.1:10808"
	return tgbotapi.NewBotAPIWithClient(t.config.Token,
		tgbotapi.APIEndpoint, client,
	)
}

func (t *Bot) Init() {
	if t.botApi != nil {
		return
	}
	// 10s 内不重试
	if t.lastErr != nil && time.Since(t.lastErrAt) < time.Second*10 {
		return
	}
	t.Lock()
	defer t.Unlock()
	if t.botApi == nil {
		bot, err := t.client()
		if err != nil {
			t.lastErrAt = time.Now()
		}
		t.botApi, t.lastErr = bot, errors.Wrap(err, "bot int failed")
	}
}

func (t *Bot) send(msg tgbotapi.Chattable) error {
	t.Init()
	if t.lastErr != nil {
		return t.lastErr
	}
	_, err := t.botApi.Send(msg)
	return err
}

func (t *Bot) SendMessage(message string) error {
	msg := tgbotapi.NewMessage(t.config.ChatId, message)
	return t.send(msg)
}

func (t *Bot) SendGroupMessage(chatId int64, message string) error {
	msg := tgbotapi.NewMessage(chatId, message)
	return t.send(msg)
}
