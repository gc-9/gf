package auth

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"fmt"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/errors"
	"io"
)

func NewEncryptService(option *config.Crypto) (*EncryptService, error) {
	cipherBlock, err := aes.NewCipher([]byte(option.Key))
	if err != nil {
		return nil, errors.Wrap(err, "NewEncryptService failed")
	}

	return &EncryptService{
		cipherBlock: cipherBlock,
	}, nil
}

type EncryptService struct {
	cipherBlock cipher.Block
}

func (t *EncryptService) Encrypt(data []byte) []byte {
	data = pkcs7Padding(data, t.cipherBlock.BlockSize())

	encrypted := make([]byte, aes.BlockSize+len(data))
	iv := encrypted[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	mode := cipher.NewCBCEncrypter(t.cipherBlock, iv)
	mode.CryptBlocks(encrypted[aes.BlockSize:], data)

	// encrypted = rand iv + crypt(dataWithPadding)
	return encrypted
}

func (t *EncryptService) Decrypt(encrypted []byte) (buf []byte, err error) {
	if len(encrypted) < aes.BlockSize*2 {
		err = errors.New("cipherText too short")
		return
	}

	defer func() {
		e := recover()
		if e != nil {
			buf = nil
			err = errors.New(fmt.Sprintf("%v", e))
		}
	}()

	iv := encrypted[:aes.BlockSize]
	buf = encrypted[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(t.cipherBlock, iv)
	mode.CryptBlocks(buf, buf)
	buf = pkcs7UnPadding(buf)

	return
}

func pkcs7Padding(src []byte, blockSize int) []byte {
	padding := blockSize - len(src)%blockSize
	padText := bytes.Repeat([]byte{byte(padding)}, padding)
	return append(src, padText...)
}

func pkcs7UnPadding(src []byte) []byte {
	length := len(src)
	unPadding := int(src[length-1])
	return src[:(length - unPadding)]
}
