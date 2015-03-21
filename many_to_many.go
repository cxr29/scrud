package scrud

import (
	"errors"
	"fmt"
	"reflect"

	"github.com/cxr29/scrud/internal/table"
	. "github.com/cxr29/scrud/query"
)

// return many to many manager of the field
func (db *DB) ManyToMany(field string, data interface{}) *ManyToMany {
	return newManyToMany(db, field, data)
}

func (tx *Tx) ManyToMany(field string, data interface{}) *ManyToMany {
	return newManyToMany(tx, field, data)
}

// many to many field manager
type ManyToMany struct {
	err    error
	xr     faker
	table  *table.Table
	column *table.Column
	value  reflect.Value
}

func errManyToMany(err error) *ManyToMany {
	return &ManyToMany{err: err}
}

func newManyToMany(xr faker, field string, data interface{}) *ManyToMany {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return errManyToMany(errors.New("scrud: many to many nil"))
		}
		v = v.Elem()
		t = v.Type()
	}
	x, err := table.TableOf(t)
	if err != nil {
		return errManyToMany(err)
	}
	if c := x.FindField(field); c == nil {
		return errManyToMany(fmt.Errorf("scrud: many to many column not found: %s/%s", x.Type.Name(), field))
	} else if c.Relation != table.ManyToMany {
		return errManyToMany(errors.New("scrud: not many to many column: " + c.FullName()))
	} else {
		return &ManyToMany{xr: xr, table: x, column: c, value: v}
	}
}

func (m2m *ManyToMany) Empty() error {
	if m2m.err != nil {
		return m2m.err
	}
	left, err := m2m.table.PrimaryKey.GetValue(m2m.value)
	if err != nil {
		return err
	}
	var q Expression
	if m2m.column.ThroughTable != nil {
		q = Delete(m2m.column.ThroughTable.Name).Where(Eq(m2m.column.ThroughLeft.Name, left))
	} else {
		q = Delete(m2m.column.Name).Where(Eq(m2m.column.NameLeft, left))
	}
	_, err = m2m.xr.Run(q)
	return err
}

func (m2m *ManyToMany) right(data interface{}) (interface{}, error) {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, errors.New("scrud: many to many nil: " + m2m.column.FullName())
		}
		v = v.Elem()
		t = v.Type()
	}
	if t != m2m.column.RelationTable.Type {
		return nil, errors.New("scrud: many to many type mismatching: " + m2m.column.FullName())
	}
	return m2m.column.RelationTable.PrimaryKey.GetValue(v)
}

// data should be the relation's type
func (m2m *ManyToMany) Has(data interface{}) (bool, error) {
	if m2m.err != nil {
		return false, m2m.err
	}

	right, err := m2m.right(data)
	if err != nil {
		return false, err
	}

	left, err := m2m.table.PrimaryKey.GetValue(m2m.value)
	if err != nil {
		return false, err
	}

	query := Select(Expr("COUNT(*)"))
	if m2m.column.ThroughTable != nil {
		query.From(m2m.column.ThroughTable.Name).Where(
			Eq(m2m.column.ThroughLeft.Name, left),
			Eq(m2m.column.ThroughRight.Name, right),
		)
	} else {
		query.From(m2m.column.Name).Where(
			Eq(m2m.column.NameLeft, left),
			Eq(m2m.column.NameRight, right),
		)
	}
	query.Limit(1)

	var count int
	q, a, err := query.Expand(m2m.xr.Starter())
	if err == nil {
		err = m2m.xr.QueryRow(q, a...).Scan(&count)
	}
	if err != nil {
		return false, err
	}
	return count == 1, nil
}

func (m2m *ManyToMany) set(empty bool, data ...interface{}) error {
	if m2m.err != nil {
		return m2m.err
	}

	left, err := m2m.table.PrimaryKey.GetValue(m2m.value)
	if err != nil {
		return err
	}

	q := Insert(m2m.column.Name).Columns(m2m.column.NameLeft, m2m.column.NameRight)
	if m2m.column.ThroughTable != nil {
		q = Insert(m2m.column.ThroughTable.Name).Columns(m2m.column.ThroughLeft.Name, m2m.column.ThroughRight.Name)
	}
	for _, i := range data {
		right, err := m2m.right(i)
		if err != nil {
			return err
		}
		q.Values(left, right)
	}

	if empty {
		if err := m2m.Empty(); err != nil {
			return err
		}
	}

	_, err = m2m.xr.Run(q)
	return err
}

// data should be the relation's type
func (m2m *ManyToMany) Set(data ...interface{}) error {
	return m2m.set(true, data...)
}

// data should be the relation's type
func (m2m *ManyToMany) Add(data ...interface{}) error {
	return m2m.set(false, data...)
}

// data should be the relation's type
func (m2m *ManyToMany) Remove(data ...interface{}) error {
	if m2m.err != nil {
		return m2m.err
	}
	left, err := m2m.table.PrimaryKey.GetValue(m2m.value)
	if err != nil {
		return err
	}

	a := make([]interface{}, 0, len(data))
	for _, i := range data {
		right, err := m2m.right(i)
		if err != nil {
			return err
		}
		a = append(a, right)
	}

	var q Expression
	if m2m.column.ThroughTable != nil {
		q = Delete(m2m.column.ThroughTable.Name).Where(
			Eq(m2m.column.ThroughLeft.Name, left),
			In(m2m.column.ThroughRight.Name, a...),
		)
	} else {
		q = Delete(m2m.column.Name).Where(
			Eq(m2m.column.NameLeft, left),
			In(m2m.column.NameRight, a...),
		)
	}
	_, err = m2m.xr.Run(q)
	return err
}
