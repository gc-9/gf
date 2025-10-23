package services

import (
	"context"
	"github.com/gc-9/gf/config"
	"github.com/gc-9/gf/crud"
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/mod/admin/types"
	"github.com/gc-9/gf/storage"
	"github.com/h2non/filetype"
	gonanoid "github.com/matoous/go-nanoid/v2"
	"github.com/samber/lo"
	"mime/multipart"
	"path"
	"strings"
	"time"
	"xorm.io/xorm"
)

type AttachmentOptions struct {
	KeyTpl    string `json:"keyTpl" yaml:"keyTpl"`
	TmpKeyTpl string `json:"tmpKeyTpl" yaml:"tmpKeyTpl"`

	tmpPrefix string
}

func NewAttachmentService(db *xorm.Engine, conf *config.Config, storage storage.Storage) (*AttachmentService, error) {
	var options AttachmentOptions
	err := conf.Get("attachmentService", &options)
	if err != nil {
		return nil, err
	}
	if options.KeyTpl == "" {
		options.KeyTpl = "files/{date}/{uuid}.{ext}"
	}
	if options.TmpKeyTpl == "" {
		options.TmpKeyTpl = "tmp_7d/{date}/{uuid}.{ext}"
	}
	options.tmpPrefix = strings.Split(options.TmpKeyTpl, "/")[0] + "/"

	return &AttachmentService{db: db,
		options: &options,
		storage: storage,
		CrudDB:  crud.NewCrudDB[types.Attachment](db),
	}, nil
}

type AttachmentService struct {
	*crud.CrudDB[types.Attachment]
	db      *xorm.Engine
	storage storage.Storage
	options *AttachmentOptions
}

var nanoidAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (t *AttachmentService) Url(path string) string {
	return t.storage.Url(path)
}

func (t *AttachmentService) Store(uid int, fh *multipart.FileHeader) (*types.AttachmentItem, error) {
	return t.StorePath(uid, fh, t.options.KeyTpl, nil)
}

func (t *AttachmentService) StoreTmp(fh *multipart.FileHeader, allowsExt []string) (*storage.FileInfo, error) {
	keyTpl := t.options.TmpKeyTpl

	f, err := fh.Open()
	if err != nil {
		return nil, errors.Wrap(err, "fh.Open failed")
	}

	ext := strings.ToLower(strings.TrimLeft(path.Ext(fh.Filename), "."))
	if ext == "" {
		f2, err := fh.Open()
		if err != nil {
			return nil, errors.Wrap(err, "fh.Open failed")
		}
		defer f2.Close()
		// get real type
		kind, _ := filetype.MatchReader(f2)
		if kind == filetype.Unknown {
			return nil, errors.New("paramError_imageTypes")
		}
		ext = kind.Extension
	}
	if len(allowsExt) > 0 && !lo.Contains(allowsExt, ext) {
		return nil, errors.New("paramError_imageTypes")
	}

	key := t.GeneratePath(keyTpl, ext)
	return t.storage.Put(context.Background(), key, f)
}

func (t *AttachmentService) GenerateTmpPath(ext string) string {
	return t.GeneratePath(t.options.TmpKeyTpl, ext)
}

func (t *AttachmentService) GeneratePath(keyTpl string, ext string) string {
	key := strings.Replace(keyTpl, "{date}", time.Now().Format("20060102"), -1)
	key = strings.Replace(key, "{uuid}", gonanoid.MustGenerate(nanoidAlphabet, 16), -1)
	key = strings.Replace(key, "{ext}", ext, -1)
	return key
}

func (t *AttachmentService) RenameTmp2Normal(key string) (string, error) {
	key = t.storage.Path(key)
	if key == "" || !strings.HasPrefix(key, t.options.tmpPrefix) {
		return key, nil
	}
	ext := strings.ToLower(strings.TrimLeft(path.Ext(key), "."))
	targetKey := t.GeneratePath(t.options.KeyTpl, ext)
	return targetKey, t.storage.Rename(context.Background(), key, targetKey)
}

func (t *AttachmentService) IsTempPath(key string) bool {
	key = t.storage.Path(key)
	if key == "" || !strings.HasPrefix(key, t.options.tmpPrefix) {
		return false
	}
	return true
}

func (t *AttachmentService) CopyTmp2Normal(key string) (string, error) {
	key = t.storage.Path(key)
	if key == "" || !strings.HasPrefix(key, t.options.tmpPrefix) {
		return key, nil
	}
	ext := strings.ToLower(strings.TrimLeft(path.Ext(key), "."))
	targetKey := t.GeneratePath(t.options.KeyTpl, ext)
	return targetKey, t.storage.Copy(context.Background(), key, targetKey)
}

func (t *AttachmentService) StorePath(uid int, fh *multipart.FileHeader, keyTpl string, allowsExt []string) (*types.AttachmentItem, error) {
	f, err := fh.Open()
	if err != nil {
		return nil, errors.Wrap(err, "fh.Open failed")
	}

	ext := strings.ToLower(strings.TrimLeft(path.Ext(fh.Filename), "."))
	if ext == "" {
		f2, err := fh.Open()
		if err != nil {
			return nil, errors.Wrap(err, "fh.Open failed")
		}
		defer f2.Close()
		// get real type
		kind, _ := filetype.MatchReader(f2)
		if kind == filetype.Unknown {
			return nil, errors.New("paramError_imageTypes")
		}
		ext = kind.Extension
	}
	if len(allowsExt) > 0 && !lo.Contains(allowsExt, ext) {
		return nil, errors.New("paramError_imageTypes")
	}

	key := t.GeneratePath(keyTpl, ext)
	finfo, err := t.storage.Put(context.Background(), key, f)
	if err != nil {
		return nil, err
	}

	m := &types.Attachment{
		UID:      uid,
		Path:     finfo.Path,
		Driver:   t.storage.Name(),
		Filename: fh.Filename,
		Size:     int(fh.Size),
		Ext:      ext,
	}
	_, err = t.Create(m)
	if err != nil {
		return nil, err
	}
	return &types.AttachmentItem{Attachment: m, Url: t.Url(m.Path)}, nil
}

// only delete db record

func (t *AttachmentService) Destroy(id int) (int, error) {
	return t.Delete(id)
}

// FullDestroy delete file and db record
func (t *AttachmentService) FullDestroy(id int) (int, error) {
	item, err := t.Get(id)
	if err != nil || item == nil {
		return 0, err
	}

	c, err := t.Delete(id)
	if err != nil {
		return 0, err
	}
	err = t.storage.Delete(context.Background(), item.Path)
	return c, err
}
