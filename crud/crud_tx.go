package crud

import (
	"github.com/gc-9/gf/types"
	"reflect"
	"xorm.io/xorm"
)

func NewCrudTX[T any](tx *xorm.Session) *CrudTX[T] {
	var t T
	kind := reflect.TypeOf(t).Kind()
	if kind == reflect.Pointer {
		panic("NewCrudDB T must not be pointer")
	}
	if kind != reflect.Struct {
		panic("NewCrudDB T must be struct")
	}

	return &CrudTX[T]{
		tx: tx,
	}
}

type CrudTX[T any] struct {
	tx *xorm.Session
}

func (s *CrudTX[T]) Get(id int, options ...QueryOption) (*T, error) {
	return GetTX[T](s.tx, id, options...)
}

func (s *CrudTX[T]) Count(options ...QueryOption) (int, error) {
	return CountTX[T](s.tx, options...)
}

func (s *CrudTX[T]) Exist(id int) (bool, error) {
	return ExistTX[T](s.tx, id)
}

func (s *CrudTX[T]) ExistByOptions(options ...QueryOption) (bool, error) {
	return ExistByOptionsTX[T](s.tx, options...)
}

func (s *CrudTX[T]) List(options ...QueryOption) ([]*T, error) {
	return ListTX[T](s.tx, options...)
}

func (s *CrudTX[T]) PagerData(pager *types.ParamPager, options ...QueryOption) (*types.PagerData[*T], error) {
	return PagerDataTX[T](s.tx, pager, options...)
}

func (s *CrudTX[T]) Create(t *T) (*T, error) {
	return CreateTX[T](s.tx, t)
}

func (s *CrudTX[T]) Creates(ts []*T) (int, error) {
	return CreatesTX[T](s.tx, ts)
}

func (s *CrudTX[T]) Update(id int, t *T, options ...QueryOption) (int, error) {
	return UpdateTX[T](s.tx, id, t, options...)
}

func (s *CrudTX[T]) Delete(id int) (int, error) {
	return DeleteTX[T](s.tx, id)
}

func (s *CrudTX[T]) DeleteOptions(options ...QueryOption) (int, error) {
	return DeleteOptionsTX[T](s.tx, options...)
}
