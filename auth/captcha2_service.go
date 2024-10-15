package auth

import (
	"bytes"
	"encoding/base64"
	"github.com/afocus/captcha"
	"github.com/gc-9/gf/errors"
	"image/color"
	"image/png"
	"strconv"
	"strings"
	"time"
)

type CaptchaProvide interface {
	Alloc(width int, height int) (*CaptchaAlloc, error)
	Validate(id string, text string) (bool, error)
}

func NewCaptcha2Service(encryptService *EncryptService) (CaptchaProvide, error) {
	return &Captcha2Service{encryptService: encryptService}, nil
}

type Captcha2Service struct {
	encryptService *EncryptService
}

func (t *Captcha2Service) Alloc(width int, height int) (*CaptchaAlloc, error) {
	cap := captcha.New()
	// 可以设置多个字体 或使用cap.AddFont("xx.ttf")追加
	cap.SetFont("resources/fonts/comic.ttf")
	// 设置验证码大小
	cap.SetSize(width, height)
	// 设置干扰强度
	cap.SetDisturbance(captcha.NORMAL)
	// 设置前景色 可以多个 随机替换文字颜色 默认黑色
	//cap.SetFrontColor(color.RGBA{0, 0, 0, 255})
	// 设置背景色 可以多个 随机替换背景色 默认白色
	cap.SetBkgColor(color.RGBA{0, 0, 0, 0})
	img, text := cap.Create(4, captcha.UPPER)

	buf := bytes.NewBuffer(nil)
	err := png.Encode(buf, img)
	if err != nil {
		return nil, err
	}

	encryptText := strconv.Itoa(int(time.Now().Unix())) + "|" + text
	tokenBuf := t.encryptService.Encrypt([]byte(encryptText))
	token := base64.StdEncoding.EncodeToString(tokenBuf)

	// imgbuf to base54
	imgBase64Str := "data:image/jpeg;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	return &CaptchaAlloc{
		ID:      token,
		Captcha: imgBase64Str,
	}, nil
}

func (t *Captcha2Service) Validate(id string, text string) (bool, error) {
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
