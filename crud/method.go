package crud

import (
	"github.com/gc-9/gf/errors"
	"github.com/gc-9/gf/types"
	"xorm.io/xorm"
)

type QueryOption func(query *xorm.Session)

var QueryForUpdate = func(query *xorm.Session) { query.ForUpdate() }

func QueryOrderBy(order string) QueryOption {
	return func(query *xorm.Session) {
		query.OrderBy(order)
	}
}

func QueryMustCols(columns ...string) QueryOption {
	return func(query *xorm.Session) {
		query.MustCols(columns...)
	}
}

func Count[T any](db *xorm.Engine, options ...QueryOption) (int, error) {
	session := db.NewSession()
	defer session.Close()

	return CountTX[T](session, options...)
}

func Exist[T any](db *xorm.Engine, id int) (bool, error) {
	session := db.NewSession()
	defer session.Close()

	return ExistTX[T](session, id)
}

func ExistByOptions[T any](db *xorm.Engine, options ...QueryOption) (bool, error) {
	session := db.NewSession()
	defer session.Close()

	return ExistByOptionsTX[T](session, options...)
}

func Get[T any](db *xorm.Engine, id int, options ...QueryOption) (*T, error) {
	session := db.NewSession()
	defer session.Close()

	return GetTX[T](session, id, options...)
}

func GetByOptions[T any](db *xorm.Engine, options ...QueryOption) (*T, error) {
	session := db.NewSession()
	defer session.Close()

	return GetByOptionsTX[T](session, options...)
}

func List[T any](db *xorm.Engine, options ...QueryOption) ([]*T, error) {
	session := db.NewSession()
	defer session.Close()

	return ListTX[T](session, options...)
}

func PagerData[T any](db *xorm.Engine, pager *types.ParamPager, options ...QueryOption) (*types.PagerData[*T], error) {
	session := db.NewSession()
	defer session.Close()

	return PagerDataTX[T](session, pager, options...)
}

func Create[T any](db *xorm.Engine, t *T) (*T, error) {
	session := db.NewSession()
	defer session.Close()

	return CreateTX[T](session, t)
}

func Creates[T any](db *xorm.Engine, ts []*T) (int, error) {
	session := db.NewSession()
	defer session.Close()

	return CreatesTX[T](session, ts)
}

// Update @up map[string]interface or struct
func Update[T any](db *xorm.Engine, id int, up interface{}, options ...QueryOption) (int, error) {
	session := db.NewSession()
	defer session.Close()

	return UpdateTX[T](session, id, up, options...)
}

func Delete[T any](db *xorm.Engine, id int) (int, error) {
	session := db.NewSession()
	defer session.Close()

	return DeleteTX[T](session, id)
}

func DeleteOptions[T any](db *xorm.Engine, options ...QueryOption) (int, error) {
	session := db.NewSession()
	defer session.Close()

	return DeleteOptionsTX[T](session, options...)
}

// tx methods
func CreateTX[T any](session *xorm.Session, t *T) (*T, error) {
	_, err := session.Insert(t)
	if err != nil {
		return nil, errors.Wrap(err, "db Insert failed")
	}
	return t, err
}

func CreatesTX[T any](session *xorm.Session, ts []*T) (int, error) {
	c, err := session.Insert(ts)
	if err != nil {
		return 0, errors.Wrap(err, "db Insert failed")
	}
	return int(c), err
}

func CountTX[T any](session *xorm.Session, options ...QueryOption) (int, error) {
	var t T
	// add options
	for _, opFunc := range options {
		opFunc(session)
	}
	c, err := session.Count(&t)
	if err != nil {
		return 0, errors.Wrap(err, "db Count failed")
	}
	return int(c), err
}

func ExistTX[T any](session *xorm.Session, id int) (bool, error) {
	var t T
	b, err := session.ID(id).Exist(&t)
	if err != nil {
		return false, errors.Wrap(err, "db Exist failed")
	}
	return b, err
}

func ExistByOptionsTX[T any](session *xorm.Session, options ...QueryOption) (bool, error) {
	var t T
	// add options
	for _, opFunc := range options {
		opFunc(session)
	}
	b, err := session.Exist(&t)
	if err != nil {
		return false, errors.Wrap(err, "db Exist failed")
	}
	return b, err
}

func GetTX[T any](session *xorm.Session, id int, options ...QueryOption) (*T, error) {
	var t T
	session.ID(id)
	// add options
	for _, opFunc := range options {
		opFunc(session)
	}
	has, err := session.Get(&t)
	if err != nil || !has {
		return nil, errors.Wrap(err, "db Get failed")
	}
	return &t, err
}

func GetByOptionsTX[T any](session *xorm.Session, options ...QueryOption) (*T, error) {
	var t T
	// add options
	for _, opFunc := range options {
		opFunc(session)
	}
	has, err := session.Get(&t)
	if err != nil || !has {
		return nil, errors.Wrap(err, "db Get failed")
	}
	return &t, err
}

func ListTX[T any](session *xorm.Session, options ...QueryOption) ([]*T, error) {

	// add options
	for _, opFunc := range options {
		opFunc(session)
	}

	var items []*T
	err := session.Find(&items)
	if err != nil {
		return nil, errors.Wrap(err, "db FindAndCount failed")
	}

	return items, nil
}

func PagerDataTX[T any](session *xorm.Session, pager *types.ParamPager, options ...QueryOption) (*types.PagerData[*T], error) {
	session.Limit(pager.PageSize, pager.Offset)

	// add options
	for _, opFunc := range options {
		opFunc(session)
	}

	var items []*T
	c, err := session.FindAndCount(&items)
	if err != nil {
		return nil, errors.Wrap(err, "db FindAndCount failed")
	}

	pagerData := &types.PagerData[*T]{List: items}
	pagerData.Pager = &types.Pager{
		PageNum:  pager.PageNum,
		PageSize: pager.PageSize,
		Total:    int(c),
	}
	return pagerData, nil
}

func UpdateTX[T any](session *xorm.Session, id int, up interface{}, options ...QueryOption) (int, error) {
	session.ID(id)
	for _, opFunc := range options {
		opFunc(session)
	}
	var t T
	c, err := session.Table(&t).Update(up)
	if err != nil {
		return 0, errors.Wrap(err, "db Update failed")
	}
	return int(c), err
}

func DeleteTX[T any](session *xorm.Session, id int) (int, error) {
	var t T
	c, err := session.ID(id).Delete(&t)
	if err != nil {
		return 0, errors.Wrap(err, "db Delete failed")
	}
	return int(c), err
}

func DeleteOptionsTX[T any](session *xorm.Session, options ...QueryOption) (int, error) {
	var t T
	for _, opFunc := range options {
		opFunc(session)
	}
	c, err := session.Delete(&t)
	if err != nil {
		return 0, errors.Wrap(err, "db Delete failed")
	}
	return int(c), err
}
