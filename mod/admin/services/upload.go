package services

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"os"
	"os/exec"
	"path"
	"path/filepath"
	"strings"

	"github.com/gc-9/gf/errors"
	"github.com/h2non/filetype"
)

// ResolveUploadExtReader resolves the upload extension and returns a reader
// that includes any bytes consumed while detecting the file type.
func ResolveUploadExtReader(filename string, reader io.Reader, forceReadRealType bool) (string, io.Reader, error) {
	ext := strings.ToLower(strings.TrimLeft(path.Ext(filename), "."))
	if ext != "" && !forceReadRealType {
		return ext, reader, nil
	}

	header := make([]byte, 8192)
	n, err := io.ReadFull(reader, header)
	if err != nil && err != io.EOF && err != io.ErrUnexpectedEOF {
		return "", nil, errors.Wrap(err, "读取文件失败")
	}
	header = header[:n]

	kind, _ := filetype.Match(header)
	if kind == filetype.Unknown {
		return "", nil, errors.New("文件类型错误")
	}
	return kind.Extension, io.MultiReader(bytes.NewReader(header), reader), nil
}

// PrepareUploadFile opens an upload and converts actual HEIC content to JPG
// when convertHeicToJpg is true, even if the uploaded filename has a .jpg suffix.
func PrepareUploadFile(fh *multipart.FileHeader, convertHeicToJpg bool) (io.Reader, func() error, string, int, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, nil, "", 0, errors.Wrap(err, "fh.Open failed")
	}

	ext, reader, err := ResolveUploadExtReader(fh.Filename, f, convertHeicToJpg)
	if err != nil {
		f.Close()
		return nil, nil, "", 0, err
	}
	if !convertHeicToJpg {
		return reader, f.Close, ext, int(fh.Size), nil
	}

	data, err := io.ReadAll(reader)
	if err != nil {
		f.Close()
		return nil, nil, "", 0, errors.Wrap(err, "读取上传文件失败")
	}

	kind, _ := filetype.Match(data)
	if kind.Extension != "heif" {
		return bytes.NewReader(data), f.Close, ext, len(data), nil
	}

	jpg, err := convertHeicToJPG(data)
	if err != nil {
		f.Close()
		return nil, nil, "", 0, err
	}
	f.Close()
	return bytes.NewReader(jpg), func() error { return nil }, "jpg", len(jpg), nil
}

func convertHeicToJPG(data []byte) ([]byte, error) {
	if _, err := exec.LookPath("heif-convert"); err != nil {
		return nil, errors.Wrap(err, "heif-convert 未安装或不在 PATH 中")
	}

	tempDir, err := os.MkdirTemp("", "gf-heic-")
	if err != nil {
		return nil, errors.Wrap(err, "创建 HEIC 转换临时目录失败")
	}
	defer os.RemoveAll(tempDir)

	inputPath := filepath.Join(tempDir, "input.heic")
	outputPath := filepath.Join(tempDir, "output.jpg")
	if err := os.WriteFile(inputPath, data, 0600); err != nil {
		return nil, errors.Wrap(err, "写入 HEIC 临时文件失败")
	}

	cmd := exec.CommandContext(context.Background(), "heif-convert", "-q", "90", inputPath, outputPath)
	var stderr bytes.Buffer
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		message := strings.TrimSpace(stderr.String())
		if message != "" {
			return nil, errors.Wrapf(err, "heif-convert 转换失败: %s", message)
		}
		return nil, errors.Wrap(err, "heif-convert 转换失败")
	}

	jpg, err := os.ReadFile(outputPath)
	if err != nil {
		return nil, errors.Wrap(err, "读取 JPG 临时文件失败")
	}
	return jpg, nil
}
