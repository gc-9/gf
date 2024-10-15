package crud

import (
	"github.com/gc-9/gf/errors"
	"xorm.io/xorm"
)

// ExecMustEffect rowsAffected must > 0
func ExecMustEffect(db *xorm.Engine, args ...interface{}) (int64, error) {
	tx := db.NewSession()
	defer tx.Close()
	return ExecMustEffectTx(tx, args)
}

// ExecMustEffectTx rowsAffected must > 0
func ExecMustEffectTx(tx *xorm.Session, args ...interface{}) (int64, error) {
	result, err := tx.Exec(args...)
	if err != nil {
		return 0, err
	}
	c, err := result.RowsAffected()
	if err != nil {
		return 0, errors.Wrap(err, "db RowsAffected failed")
	}
	if c <= 0 {
		return 0, errors.WithStack("db no RowsAffected")
	}
	return c, nil
}
