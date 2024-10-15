package telegram

import (
	"log"
	"sync"
)

func NewBotLogger(bot *Bot, maxQueueSize int) *BotLogger {
	return &BotLogger{
		bot:          bot,
		maxQueueSize: maxQueueSize,
		q:            make([][]byte, 0),
		lock:         sync.Mutex{},
	}
}

type BotLogger struct {
	bot          *Bot
	q            [][]byte
	lock         sync.Mutex
	maxQueueSize int
	isStart      bool
}

func (t *BotLogger) Write(p []byte) (n int, err error) {
	defer t.onceStart()

	t.lock.Lock()
	// need clone []byte
	p = append([]byte{}, p...)
	t.q = append(t.q, p)
	if len(t.q) > t.maxQueueSize {
		t.q = t.q[1:]
		log.Printf("ERROR [BotLogger] queue is full: %v", t.maxQueueSize)
	}
	t.lock.Unlock()
	return len(p), nil
}

func (t *BotLogger) Sync() error {
	return nil
}

func (t *BotLogger) getOne() []byte {
	t.lock.Lock()
	defer t.lock.Unlock()
	if len(t.q) == 0 {
		return nil
	}
	m := t.q[0]
	t.q = t.q[1:]
	return m
}

func (t *BotLogger) onceStart() {
	if !t.isStart {
		t.isStart = true
		go t.startSend()
	}
}

func (t *BotLogger) startSend() {
	for {
		m := t.getOne()
		if m == nil {
			t.isStart = false
			return
		}
		err := t.bot.SendMessage(string(m))
		if err != nil {
			log.Println("ERROR [BotLogger] err send message:", err)
		}
	}
}
