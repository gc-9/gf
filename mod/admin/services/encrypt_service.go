package services

import (
	"bytes"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"io"
)

func NewEncryptService(key string) *EncryptService {
	cipherBlock, err := aes.NewCipher([]byte(key))
	if err != nil {
		panic(err)
	}

	return &EncryptService{
		cipherBlock: cipherBlock,
	}
}

type EncryptService struct {
	cipherBlock cipher.Block
}

func (t *EncryptService) AesEcbEncrypt(data []byte) []byte {
	data = pkcs7Padding(data, t.cipherBlock.BlockSize())

	encrypted := make([]byte, aes.BlockSize+len(data))
	iv := encrypted[:aes.BlockSize]
	if _, err := io.ReadFull(rand.Reader, iv); err != nil {
		panic(err)
	}

	mode := cipher.NewCBCEncrypter(t.cipherBlock, iv)
	mode.CryptBlocks(encrypted[aes.BlockSize:], data)

	return encrypted
}

func (t *EncryptService) AesCbcDecrypt(encrypted []byte) []byte {

	iv := encrypted[:aes.BlockSize]
	encrypted = encrypted[aes.BlockSize:]

	mode := cipher.NewCBCDecrypter(t.cipherBlock, iv)
	mode.CryptBlocks(encrypted, encrypted)

	decrypted := pkcs7UnPadding(encrypted)
	return decrypted
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
