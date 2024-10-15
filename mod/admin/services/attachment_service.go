package services

import (
	"context"
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

func NewAttachmentService(db *xorm.Engine, prefix string, storage storage.Storage) *AttachmentService {
	prefix = strings.Trim(prefix, "/")
	return &AttachmentService{db: db, prefix: prefix,
		storage: storage,
		CrudDB:  crud.NewCrudDB[types.Attachment](db),
	}
}

type AttachmentService struct {
	*crud.CrudDB[types.Attachment]
	prefix  string
	db      *xorm.Engine
	storage storage.Storage
}

var nanoidAlphabet = "0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ"

func (t *AttachmentService) generatePath(ext string) string {
	id := gonanoid.MustGenerate(nanoidAlphabet, 16)
	var p string
	if ext == "" {
		p = id
	} else {
		p = id + "." + ext
	}

	_, isLocal := t.storage.(*storage.Local)
	if isLocal {
		p = time.Now().Format("20060102") + "/" + p
	}

	if t.prefix != "" {
		p = t.prefix + "/" + p
	}
	return p
}

func (t *AttachmentService) Url(path string) string {
	return t.storage.Url(path)
}

func (t *AttachmentService) Store(uid int, fh *multipart.FileHeader) (*types.AttachmentItem, error) {
	ext := strings.TrimLeft(path.Ext(fh.Filename), ".")
	key := t.generatePath(ext)
	return t.StorePath(uid, fh, key, nil)
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

	key := strings.Replace(keyTpl, "{ext}", ext, -1)
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
