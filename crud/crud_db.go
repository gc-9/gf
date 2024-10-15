package crud

import (
	"github.com/gc-9/gf/types"
	"reflect"
	"xorm.io/xorm"
)

func NewCrudDB[T any](db *xorm.Engine) *CrudDB[T] {
	var t T
	kind := reflect.TypeOf(t).Kind()
	if kind == reflect.Pointer {
		panic("NewCrudDB T must not be pointer")
	}
	if kind != reflect.Struct {
		panic("NewCrudDB T must be struct")
	}

	return &CrudDB[T]{
		db: db,
	}
}

type CrudDB[T any] struct {
	db *xorm.Engine
}

func (s *CrudDB[T]) TX(session *xorm.Session) *CrudTX[T] {
	return &CrudTX[T]{tx: session}
}

func (s *CrudDB[T]) Create(t *T) (*T, error) {
	return Create[T](s.db, t)
}

func (s *CrudDB[T]) Creates(ts []*T) (int, error) {
	return Creates[T](s.db, ts)
}

func (s *CrudDB[T]) Get(id int, options ...QueryOption) (*T, error) {
	return Get[T](s.db, id, options...)
}

func (s *CrudDB[T]) GetByOptions(options ...QueryOption) (*T, error) {
	return GetByOptions[T](s.db, options...)
}

func (s *CrudDB[T]) Count(options ...QueryOption) (int, error) {
	return Count[T](s.db, options...)
}

func (s *CrudDB[T]) Exist(id int) (bool, error) {
	return Exist[T](s.db, id)
}

func (s *CrudDB[T]) ExistByOptions(options ...QueryOption) (bool, error) {
	return ExistByOptions[T](s.db, options...)
}

func (s *CrudDB[T]) List(options ...QueryOption) ([]*T, error) {
	return List[T](s.db, options...)
}

func (s *CrudDB[T]) PagerData(pager *types.ParamPager, options ...QueryOption) (*types.PagerData[*T], error) {
	return PagerData[T](s.db, pager, options...)
}

// Update @up map[string]interface or struct
func (s *CrudDB[T]) Update(id int, up any, options ...QueryOption) (int, error) {
	return Update[T](s.db, id, up, options...)
}

func (s *CrudDB[T]) Delete(id int) (int, error) {
	return Delete[T](s.db, id)
}

func (s *CrudDB[T]) DeleteOptions(options ...QueryOption) (int, error) {
	return DeleteOptions[T](s.db, options...)
}
