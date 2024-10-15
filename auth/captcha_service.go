package auth

import (
	"bytes"
	"encoding/base64"
	"github.com/gc-9/gf/errors"
	"github.com/lifei6671/gocaptcha"
	"math/rand"
	"strconv"
	"strings"
	"time"
)

func NewCaptchaService(encryptService *EncryptService) (CaptchaProvide, error) {
	// set fonts dir
	err := gocaptcha.ReadFonts("resources/fonts", ".ttf")
	if err != nil {
		return nil, err
	}
	return &CaptchaService{encryptService: encryptService}, nil
}

type CaptchaService struct {
	encryptService *EncryptService
}

type CaptchaAlloc struct {
	ID      string `json:"id"`
	Captcha string `json:"captcha"`
}

var txtChars = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"

func randText(num int) string {
	textNum := len(txtChars)
	text := ""
	r := rand.New(rand.NewSource(time.Now().UnixNano()))

	for i := 0; i < num; i++ {
		text = text + string(txtChars[r.Intn(textNum)])
	}
	return text
}

func (t *CaptchaService) Alloc(width int, height int) (*CaptchaAlloc, error) {
	text := randText(4)
	captchaImage := gocaptcha.New(width, height, gocaptcha.RandLightColor())
	err := captchaImage.
		DrawNoise(gocaptcha.CaptchaComplexLower).
		//DrawTextNoise(gocaptcha.CaptchaComplexLower).
		DrawText(text).Error
	//DrawBorder(gocaptcha.ColorToRGB(0x17A7A7A)).Error
	//DrawSineLine().Error
	if err != nil {
		return nil, err
	}

	buffer := new(bytes.Buffer)
	err = captchaImage.SaveImage(buffer, gocaptcha.ImageFormatJpeg)
	if err != nil {
		return nil, err
	}

	encryptText := strconv.Itoa(int(time.Now().Unix())) + "|" + text
	tokenBuf := t.encryptService.Encrypt([]byte(encryptText))
	token := base64.StdEncoding.EncodeToString(tokenBuf)

	// imgbuf to base54
	imgBase64Str := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buffer.Bytes())

	return &CaptchaAlloc{
		ID:      token,
		Captcha: imgBase64Str,
	}, nil
}

func (t *CaptchaService) Validate(id string, text string) (bool, error) {
	tokenBuf, err := base64.StdEncoding.DecodeString(id)
	if err != nil {
		return false, nil
	}

	v, _ := t.encryptService.Decrypt(tokenBuf)
	group := strings.Split(string(v), "|")
	if len(group) != 2 {
		return false, errors.New("验证码已过期")
	}

	beginTime, _ := strconv.Atoi(group[0])
	if time.Now().Unix()-int64(beginTime) > 60*5 {
		return false, nil
	}

	if strings.EqualFold(text, group[1]) {
		return true, nil
	}
	return false, nil
}
