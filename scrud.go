// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Go struct/SQL CRUD
//
//  import "github.com/cxr29/scrud"
//  import _ "github.com/go-sql-driver/mysql"
//
//  db, err := scrud.Open("mysql", "user:password@/database")
//
//  // A, B is struct or *struct
//  n, err := db.Insert(A)                // insert
//  n, err = db.Insert([]A{})             // batch insert
//  err = db.Select(&A, ...)              // select by primary key, support include or exclude columns
//  err = db.SelectRelation("B", &A, ...) // select relation field, support include or exclude columns
//  err = db.Update(A, ...)               // update by primary key, support include or exclude columns
//  err = db.Delete(A)                    // delete by primary key
//
//  m2m := db.ManyToMany("B", A) // many to many field manager
//  err = m2m.Add(B, ...)        // add relation
//  err = m2m.Set(B, ...)        // set relation, empty other
//  err = m2m.Remove(B, ...)     // remove relation
//  has, err := m2m.Has(B)       // check relation
//  err = m2m.Empty()            // empty relation
//
//  result, err := db.Run(qe)      // run a query expression that doesn't return rows
//  err = db.Fetch(qe).One(&A)     // run a query expression and fetch one row to struct
//  err = db.Fetch(qe).All(&[]A{}) // run a query expression and fetch rows to slice of struct
//
// See https://github.com/cxr29/scrud for more details
package scrud

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"
	"time"

	"github.com/cxr29/scrud/internal/table"
	. "github.com/cxr29/scrud/query"
)

var ErrNoRows = sql.ErrNoRows

func starter(s string) Starter {
	switch s {
	case "mysql":
		return new(MySQL)
	case "postgres":
		return new(Postgres)
	case "sqlite":
		return new(Sqlite)
	}
	return nil
}

func insert(xr faker, data interface{}) (int64, error) {
	cnt, ptr := -1, false

	v := reflect.ValueOf(data)
	t := v.Type()
	k := v.Kind()
	if k == reflect.Ptr {
		if v.IsNil() {
			return 0, errors.New("scrud: insert nil")
		}
		v = v.Elem()
		t = v.Type()
		k = v.Kind()
	}

	if k == reflect.Array || k == reflect.Slice {
		cnt = v.Len()
		if cnt == 0 {
			return 0, errors.New("scrud: empty batch insert")
		}
		t = t.Elem()
		if t.Kind() == reflect.Ptr {
			ptr = true
			t = t.Elem()
		}
	}

	x, err := table.TableOf(t)
	if err != nil {
		return 0, err
	}

	i := Insert(x.Name)

	cols := make([]string, 0)
	for _, c := range x.Columns {
		if c.IsManyRelation() || c.AutoIncrement() {
			continue
		}
		cols = append(cols, c.Name)
	}
	if len(cols) == 0 {
		return 0, errors.New("scrud: insert no columns: " + x.Type.Name())
	}

	i.Columns(cols...)

	tile := func(v reflect.Value) ([]interface{}, error) {
		now := time.Unix(time.Now().Unix(), 0)
		a := make([]interface{}, 0, len(x.ColumnMap))
		for _, c := range x.Columns {
			if c.IsManyRelation() || c.AutoIncrement() {
				continue
			}
			if c.AutoNowAdd() || c.AutoNow() {
				if err := c.SetValue(v, now); err != nil { // cxr? defer set value
					return nil, err
				} else {
					a = append(a, now)
				}
			} else if i, err := c.GetValue(v); err != nil {
				return nil, err
			} else {
				a = append(a, i)
			}
		}
		return a, nil
	}

	if cnt == -1 {
		if a, err := tile(v); err != nil {
			return 0, err
		} else {
			i.Values(a...)
		}
	} else {
		for j := 0; j < cnt; j++ {
			w := v.Index(j)
			if ptr {
				if w.IsNil() {
					return 0, errors.New("scrud: batch insert nil: " + x.Type.Name())
				}
				w = w.Elem()
			}
			if a, err := tile(w); err != nil {
				return 0, err
			} else {
				i.Values(a...)
			}
		}
	}

	q, a, err := i.Expand(xr.Starter())
	if err != nil {
		return 0, err
	}

	r, err := xr.Exec(q, a...)
	if err != nil {
		return 0, err
	}

	if cnt == -1 && x.AutoIncrement != nil {
		ai, err := r.LastInsertId()
		if err == nil {
			err = x.AutoIncrement.SetValue(v, ai)
		}
		if err != nil {
			return 0, err
		}
	}

	return r.RowsAffected()
}

func tidyColumns(action string, x *table.Table, columns ...string) (map[int]struct{}, bool, error) {
	columnMap := make(map[int]struct{})
	exclude := false
	include := "include"
	if len(columns) > 0 && columns[0] == "-" {
		exclude = true
		include = "exclude"
		columns = columns[1:]
	}
	for _, i := range columns {
		if c := x.FindField(i); c != nil {
			if c.IsManyRelation() {
				return nil, false, fmt.Errorf("scrud: %s %s many relation column: %s", action, include, c.FullName())
			} else {
				columnMap[c.Index] = struct{}{}
			}
		} else {
			return nil, false, fmt.Errorf("scrud: %s %s column not found: %s/%s", action, include, x.Type.Name(), i)
		}
	}
	return columnMap, exclude, nil
}

func selectRelation(xr faker, field string, data interface{}, columns ...string) error {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() != reflect.Ptr {
		return errors.New("scrud: select relation need pointer")
	}
	if v.IsNil() {
		return errors.New("scrud: select relation nil")
	}
	v = v.Elem()
	t = v.Type()

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	if c := x.FindField(field); c == nil {
		return fmt.Errorf("scrud: select relation column not found: %s/%s", x.Type.Name(), field)
	} else if c.IsOneRelation() {
		v = v.Field(c.Index)
		if v.Kind() != reflect.Ptr {
			v = v.Addr()
		} else if v.IsNil() {
			return errors.New("scrud: select relation nil: " + c.FullName())
		}
		return retrieve(xr, v.Interface(), columns...)
	} else if c.IsManyRelation() {
		pk, err := x.PrimaryKey.GetValue(v)
		if err != nil {
			return err
		}

		columnMap, exclude, err := tidyColumns("select relation", c.RelationTable, columns...)
		if err != nil {
			return err
		}
		count := len(columnMap)

		elect := make([]interface{}, 0)
		for _, rc := range c.RelationTable.Columns {
			if rc.IsManyRelation() {
				continue
			}
			if count > 0 {
				if _, ok := columnMap[rc.Index]; (ok && exclude) || (!ok && !exclude) {
					continue
				}
			}
			elect = append(elect, rc.Name)
		}
		if len(elect) == 0 {
			return errors.New("scrud: select relation no columns: " + c.FullName())
		}

		v = v.Field(c.Index)
		if v.Kind() != reflect.Ptr {
			v = v.Addr()
		} else if v.IsNil() {
			v.Set(reflect.New(v.Type().Elem()))
		}

		if c.Relation == table.OneToMany {
			return xr.Fetch(Select(elect...).From(c.RelationTable.Name).Where(Eq(c.Name, pk))).All(v.Interface())
		} else {
			var q Expression
			if c.ThroughTable != nil {
				q = Select(c.ThroughRight.Name).From(c.ThroughTable.Name).Where(Eq(c.ThroughLeft.Name, pk))
			} else {
				q = Select(c.NameRight).From(c.Name).Where(Eq(c.NameLeft, pk))
			}
			return xr.Fetch(Select(elect...).From(c.RelationTable.Name).Where(
				Cond("`"+c.RelationTable.PrimaryKey.Name+"` IN ?", q),
			)).All(v.Interface())
		}
	} else {
		return errors.New("scrud: select relation column no relation: " + c.FullName())
	}
}

func retrieve(xr faker, data interface{}, columns ...string) error {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() != reflect.Ptr {
		return errors.New("scrud: select need pointer")
	}
	if v.IsNil() {
		return errors.New("scrud: select nil")
	}
	v = v.Elem()
	t = v.Type()

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	if x.PrimaryKey == nil {
		return errors.New("scrud: select no primary_key: " + x.Type.Name())
	}
	pk, err := x.PrimaryKey.GetValue(v)
	if err != nil {
		return err
	}

	columnMap, exclude, err := tidyColumns("select", x, columns...)
	if err != nil {
		return err
	}
	count := len(columnMap)

	elect := make([]interface{}, 0)
	set := make(map[int]*table.Column)
	scans := make([]interface{}, 0)
	for _, c := range x.Columns {
		if c.IsManyRelation() || c.PrimaryKey() {
			continue
		}
		if count > 0 {
			if _, ok := columnMap[c.Index]; (ok && exclude) || (!ok && !exclude) {
				continue
			}
		}
		elect = append(elect, c.Name)
		if c.HasSetter() {
			set[len(scans)] = c
		}
		scans = append(scans, c.Scan(v))
	}
	if len(elect) == 0 {
		return errors.New("scrud: select no columns: " + x.Type.Name())
	}

	q, a, err := Select(elect...).From(x.Name).Where(Eq(x.PrimaryKey.Name, pk)).Expand(xr.Starter())
	if err != nil {
		return err
	}

	if err := xr.QueryRow(q, a...).Scan(scans...); err != nil {
		return err
	}

	for i, c := range set {
		if err := c.SetValue(v, reflect.ValueOf(scans[i]).Elem().Interface()); err != nil {
			return err
		}
	}

	return nil
}

func update(xr faker, data interface{}, columns ...string) error {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return errors.New("scrud: update nil")
		}
		v = v.Elem()
		t = v.Type()
	}

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	if x.PrimaryKey == nil {
		return errors.New("scrud: update no primary key: " + x.Type.Name())
	}
	pk, err := x.PrimaryKey.GetValue(v)
	if err != nil {
		return err
	}

	u := Update(x.Name).Where(Eq(x.PrimaryKey.Name, pk))

	columnMap, exclude, err := tidyColumns("update", x, columns...)
	if err != nil {
		return err
	}
	count := len(columnMap)

	for _, c := range x.Columns {
		if c.IsManyRelation() || c.PrimaryKey() {
			continue
		}
		if c.AutoNow() { // cxr? always update
			now := time.Unix(time.Now().Unix(), 0)
			if err := c.SetValue(v, now); err != nil {
				return err
			}
			u.Set(c.Name, now)
		} else {
			if count > 0 {
				if _, ok := columnMap[c.Index]; (ok && exclude) || (!ok && !exclude) {
					continue
				}
			}
			if i, err := c.GetValue(v); err != nil {
				return err
			} else {
				u.Set(c.Name, i)
			}
		}
	}

	q, a, err := u.Expand(xr.Starter())
	if err == nil {
		_, err = xr.Exec(q, a...)
	}

	return err
}

func delete(xr faker, data interface{}) error {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return errors.New("scrud: delete nil")
		}
		v = v.Elem()
		t = v.Type()
	}

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	if x.PrimaryKey == nil {
		return errors.New("scrud: delete no primary key: " + x.Type.Name())
	}
	pk, err := x.PrimaryKey.GetValue(v)
	if err != nil {
		return err
	}

	q, a, err := Delete(x.Name).Where(Eq(x.PrimaryKey.Name, pk)).Expand(xr.Starter())
	if err == nil {
		_, err = xr.Exec(q, a...)
	}

	return err
}

func fetch(xr faker, query Expression) *Rows {
	var rows *sql.Rows
	var cols []string
	q, a, err := query.Expand(xr.Starter())
	if err == nil {
		rows, err = xr.Query(q, a...)
		if err == nil {
			cols, err = rows.Columns()
		}
	}
	if err != nil {
		return &Rows{err: err}
	}
	return &Rows{Rows: rows, cols: cols}
}

func run(xr faker, query Expression) (sql.Result, error) {
	q, a, err := query.Expand(xr.Starter())
	if err != nil {
		return nil, err
	}
	return xr.Exec(q, a...)
}

type DB struct {
	*sql.DB
	driverName string
}

func Open(driverName, dataSourceName string) (*DB, error) {
	switch driverName {
	case "mysql", "postgres", "sqlite":
	default:
		return nil, fmt.Errorf("scrud: unsupported driver: %s", driverName)
	}
	db, err := sql.Open(driverName, dataSourceName)
	if err != nil {
		return nil, err
	}
	return &DB{db, driverName}, nil
}

func (db *DB) Begin() (*Tx, error) {
	tx, err := db.DB.Begin()
	if err != nil {
		return nil, err
	}
	return &Tx{tx, db.driverName}, nil
}

// return a starter to expand query expression
func (db *DB) Starter() Starter {
	return starter(db.driverName)
}

// insert struct, if have auto column data must be *struct
//
// batch insert if data is array or slice and not set auto increment column
func (db *DB) Insert(data interface{}) (int64, error) {
	return insert(db, data)
}

// select by primary key, data must be *struct
//
// columns specify which to retrieve, to exclude put minus sign at the fisrt
func (db *DB) Select(data interface{}, columns ...string) error {
	return retrieve(db, data, columns...)
}

// select relation field, data must be *struct
//
// columns specify which relation's to retrieve, to exclude put minus sign at the fisrt
func (db *DB) SelectRelation(field string, data interface{}, columns ...string) error {
	return selectRelation(db, field, data, columns...)
}

// update by primary key, if have auto now column data must be *struct
//
// columns specify which to update, to exclude put minus sign at the fisrt
func (db *DB) Update(data interface{}, columns ...string) error {
	return update(db, data, columns...)
}

// delete by primary key, data must be struct or *struct
func (db *DB) Delete(data interface{}) error {
	return delete(db, data)
}

// return many to many manager of the field
func (db *DB) ManyToMany(field string, data interface{}) *ManyToMany {
	return newManyToMany(db, field, data)
}

// fetch run a query expression that return rows, typically a select
func (db *DB) Fetch(query Expression) *Rows {
	return fetch(db, query)
}

// run a query expression that doesn't return rows, such as insert, update and delete
func (db *DB) Run(query Expression) (sql.Result, error) {
	return run(db, query)
}

type Tx struct {
	*sql.Tx
	driverName string
}

func (tx *Tx) Starter() Starter {
	return starter(tx.driverName)
}

func (tx *Tx) Insert(data interface{}) (int64, error) {
	return insert(tx, data)
}

func (tx *Tx) Select(data interface{}, columns ...string) error {
	return retrieve(tx, data, columns...)
}

func (tx *Tx) SelectRelation(field string, data interface{}, columns ...string) error {
	return selectRelation(tx, field, data, columns...)
}

func (tx *Tx) Update(data interface{}, columns ...string) error {
	return update(tx, data, columns...)
}

func (tx *Tx) Delete(data interface{}) error {
	return delete(tx, data)
}

func (tx *Tx) ManyToMany(field string, data interface{}) *ManyToMany {
	return newManyToMany(tx, field, data)
}

func (tx *Tx) Fetch(query Expression) *Rows {
	return fetch(tx, query)
}

func (tx *Tx) Run(query Expression) (sql.Result, error) {
	return run(tx, query)
}

type Rows struct {
	err error
	*sql.Rows
	cols []string
}

func (r *Rows) Err() error {
	if r.err != nil {
		return r.err
	}

	return r.Rows.Err()
}

func (r *Rows) columns(x *table.Table) ([]*table.Column, error) {
	m := make(map[int]struct{})
	cols := make([]*table.Column, 0, len(r.cols))
	for _, i := range r.cols {
		c := x.FindColumn(i)
		if c == nil {
			return nil, fmt.Errorf("scrud: scan column not found: %s/%s", x.Type.Name(), i)
		}
		if c.IsManyRelation() {
			return nil, errors.New("scrud: scan many relation column: " + c.FullName())
		}
		if _, ok := m[c.Index]; ok {
			return nil, errors.New("scrud: scan column repeat: " + c.FullName())
		} else {
			m[c.Index] = struct{}{}
		}
		cols = append(cols, c)
	}
	return cols, nil
}

// scan one row to struct
func (r *Rows) Scan(i interface{}) error {
	if r.err != nil {
		return r.err
	}

	v := reflect.ValueOf(i)
	t := v.Type()
	if t.Kind() != reflect.Ptr {
		return errors.New("scrud: scan need pointer")
	}
	t = t.Elem()
	if t.Kind() != reflect.Struct {
		return errors.New("scrud: scan need struct")
	}

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	cols, err := r.columns(x)
	if err != nil {
		return err
	}

	if v.IsNil() {
		v.Set(reflect.New(t))
	}
	v = v.Elem()

	return r.scan(cols, v)
}

func (r *Rows) scan(cols []*table.Column, v reflect.Value) error {
	set := make(map[int]*table.Column)
	scans := make([]interface{}, 0, len(cols))
	for _, c := range cols {
		if c.HasSetter() {
			set[len(scans)] = c
		}
		scans = append(scans, c.Scan(v))
	}

	if err := r.Rows.Scan(scans...); err != nil {
		return err
	}

	for i, c := range set {
		if err := c.SetValue(v, reflect.ValueOf(scans[i]).Elem().Interface()); err != nil {
			return err
		}
	}

	return nil
}

// scan one row to struct then close the rows
func (r *Rows) One(i interface{}) error {
	if r.err != nil {
		return r.err
	}

	defer r.Close()

	if r.Next() {
		return r.Scan(i)
	} else {
		return ErrNoRows
	}
}

// scan rows to slice of struct then close the rows
func (r *Rows) All(i interface{}) (err error) {
	if r.err != nil {
		return r.err
	}

	defer r.Close()

	ptr := false

	v := reflect.ValueOf(i)
	t := v.Type()

	if t.Kind() != reflect.Ptr {
		return errors.New("scrud: all need pointer")
	} else if v.IsNil() {
		return errors.New("scrud: all nil")
	} else {
		t = t.Elem()
	}

	if t.Kind() != reflect.Slice {
		return errors.New("scrud: all need slice of struct")
	} else {
		t = t.Elem()
	}

	if t.Kind() == reflect.Ptr {
		ptr = true
		t = t.Elem()
	}
	if t.Kind() != reflect.Struct {
		return errors.New("scrud: all need slice of struct")
	}

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	cols, err := r.columns(x)
	if err != nil {
		return err
	}

	v = v.Elem()
	if v.Len() != 0 {
		v.SetLen(0)
	}

	for r.Next() {
		j := reflect.New(t).Elem()
		if err := r.scan(cols, j); err != nil {
			return err
		}
		if ptr {
			j = j.Addr()
		}
		v.Set(reflect.Append(v, j))
	}

	return r.Err()
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

type faker interface {
	Exec(string, ...interface{}) (sql.Result, error)
	Query(string, ...interface{}) (*sql.Rows, error)
	QueryRow(string, ...interface{}) *sql.Row
	Starter() Starter
	Fetch(Expression) *Rows
	Run(Expression) (sql.Result, error)
}
