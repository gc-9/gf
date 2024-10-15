package util

import (
	"math"
	"math/rand"
	"strings"
)

const charsets = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const charsets_upper = "ABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
const CharsetsNumber = "0123456789"

// 生成随机字符串
func RandString(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charsets[rand.Intn(len(charsets))]
	}
	return string(b)
}

func RandStringCharsets(charsets string, length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charsets[rand.Intn(len(charsets))]
	}
	return string(b)
}

func RandStringUpper(length int) string {
	b := make([]byte, length)
	for i := range b {
		b[i] = charsets_upper[rand.Intn(len(charsets_upper))]
	}
	return string(b)
}

// Substring by chars length
func Substring(s string, start int, end int) string {
	if len(s) == 0 {
		return ""
	}
	l := end - start
	if l <= 0 {
		return ""
	}

	var charLen, iBegin, iEnd int
	for iEnd = range s {
		if charLen == start {
			iBegin = iEnd
		}
		if charLen == l {
			break
		}
		charLen++
	}

	if charLen < l {
		return s
	}
	return s[iBegin:iEnd]
}

// SubUtf8Bytes return a valid []byte
// warning!! only support utf-8 encode
// Unicode符号范围     |        UTF-8编码方式
// (十六进制)         |              （二进制）
// 0000 0000-0000 007F | 0xxxxxxx
// 0000 0080-0000 07FF | 110xxxxx 10xxxxxx
// 0000 0800-0000 FFFF | 1110xxxx 10xxxxxx 10xxxxxx
// 0001 0000-0010 FFFF | 11110xxx 10xxxxxx 10xxxxxx 10xxxxxx
func SubUtf8Bytes(buf []byte, l int) []byte {
	const l1 = byte(0b0)
	const l2 = byte(0b11 << 6)
	const l3 = byte(0b111 << 5)
	const l4 = byte(0b1111 << 4)

	const o1 = byte(0b1 << 7)
	const o5 = byte(0b11111 << 3)

	if l <= 0 {
		return []byte{}
	}
	if len(buf) <= l {
		return buf
	}

	sbuf := buf[:l]

	offset := 0
	cl := 0

Loop:
	for {
		offset++
		if l-offset < 0 {
			return []byte{}
		}
		b := sbuf[l-offset]

		switch {
		case b&o1 == l1:
			cl = 1
			break Loop
		case b&l3 == l2:
			cl = 2
			break Loop
		case b&l4 == l3:
			cl = 3
			break Loop
		case b&o5 == l4:
			cl = 4
			break Loop
		}
	}

	if cl != offset {
		sbuf = sbuf[:l-offset]
	}
	return sbuf
}

func HideSome(s string, percent float32) string {
	if percent <= 0 || percent >= 1 {
		return ""
	}

	rues := []rune(s)
	l := len(rues)
	if l == 0 {
		return s
	}
	w := int(float32(l) * percent)

	begin := int(math.Ceil(float64(l-w)) / 2)
	return string(rues[:begin]) + strings.Repeat("*", w) + string(rues[begin+w:])
}
