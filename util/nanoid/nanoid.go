package nanoid

import (
	gonanoid "github.com/matoous/go-nanoid/v2"
)

var nanoidAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func GameUuid() string {
	return gonanoid.MustGenerate(nanoidAlphabet, 10)
}
