// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package table

import (
	"database/sql"
	"database/sql/driver"
	"errors"
	"fmt"
	"reflect"
	"strings"
	"sync"
	"time"

	"github.com/cxr29/scrud/format"
)

type Tabler interface {
	// return table name
	TableName() string
}

type Columner interface {
	// field name
	// if relation's struct name, primary key field name, table name and primary key column name
	// return column name
	ColumnName(...string) string
}

type Througher interface {
	// field name
	// return through table, left field name, right field name
	ThroughTable(string) (interface{}, string, string)
}

type Table struct {
	Type          reflect.Type
	Value         reflect.Value
	Name          string
	Columns       []*Column
	FieldMap      map[string]*Column
	ColumnMap     map[string]*Column // no many relation columns
	PrimaryKey    *Column
	AutoIncrement *Column
	AutoNowAdd    *Column
	AutoNow       *Column
}

// field name then column name
func (t *Table) FindField(s string) *Column {
	if c, ok := t.FieldMap[s]; ok {
		return c
	}
	if c, ok := t.ColumnMap[s]; ok {
		return c
	}
	return nil
}

// column name then field name
func (t *Table) FindColumn(s string) *Column {
	if c, ok := t.ColumnMap[s]; ok {
		return c
	}
	if c, ok := t.FieldMap[s]; ok {
		return c
	}
	return nil
}

type Column struct {
	Table              *Table
	Type               reflect.Type
	Index              int
	Field              string
	Name               string // many_to_many is table name
	Relation           int
	RelationTable      *Table
	Valuer, Scanner    bool
	Getter, Setter     int
	GetType, SetType   reflect.Type
	GetError, SetError bool
	GetPointer         bool
	// many_to_many only
	NameLeft, NameRight       string
	ThroughTable              *Table
	ThroughLeft, ThroughRight *Column
}

func (c *Column) init(x map[reflect.Type]*Table) error {
	if c.Relation != 0 && (c.HasGetter() || c.HasSetter()) {
		return errors.New("table: relation field not allow getter and setter: " + c.FullName())
	}

	switch c.Relation {
	case OneToMany, ManyToMany:
		if c.Table.PrimaryKey == nil {
			return errors.New("table: relation struct need primary_key: " + c.Table.Type.Name())
		}
	}

	rt := c.RelationTable
	switch c.Relation {
	case OneToOne, ManyToOne, ManyToMany:
		if rt.PrimaryKey == nil {
			return errors.New("table: relation struct need primary_key: " + rt.Type.Name())
		}
	}

	columner, _ := c.Table.Value.Interface().(Columner)
	througher, _ := c.Table.Value.Interface().(Througher)

	if !isManyRelation(c.Relation) {
		if c.Name == "" {
			if columner != nil {
				if c.Relation != 0 {
					c.Name = columner.ColumnName(c.Field,
						rt.Type.Name(), rt.PrimaryKey.Field, rt.Name, rt.PrimaryKey.Name)
				} else {
					c.Name = columner.ColumnName(c.Field)
				}
			} else if c.Relation != 0 {
				c.Name = format.ColumnName(c.Field, c.Table.Type.Name(), c.Table.Name,
					rt.Type.Name(), rt.PrimaryKey.Field, rt.Name, rt.PrimaryKey.Name)
			} else {
				c.Name = format.ColumnName(c.Field, c.Table.Type.Name(), c.Table.Name)
			}
		}

		if _, ok := c.Table.ColumnMap[c.Name]; ok {
			return errors.New("table: column name repeat: " + c.FullName())
		} else {
			c.Table.ColumnMap[c.Name] = c
		}
	} else if c.Relation == OneToMany {
		if c.Name == "" {
			if columner != nil {
				c.Name = columner.ColumnName(c.Field,
					c.Table.Type.Name(), c.Table.PrimaryKey.Field, c.Table.Name, c.Table.PrimaryKey.Name)
			} else {
				c.Name = format.ColumnName(c.Field, c.Table.Type.Name(), c.Table.Name,
					c.Table.Type.Name(), c.Table.PrimaryKey.Field, c.Table.Name, c.Table.PrimaryKey.Name,
				)
			}
		}
	} else if c.Relation == ManyToMany {
		if c.Name != "" {
			if i := strings.Index(c.Name, "|"); i != -1 {
				c.NameLeft = c.Name[i+1:]
				c.Name = c.Name[:i]
				if i := strings.Index(c.NameLeft, "|"); i != -1 {
					c.NameRight = c.NameLeft[i+1:]
					c.NameLeft = c.NameLeft[:i]
				}
			}
		}

		if c.Name == "" {
			c.Name = format.ManyToManyTableName(c.Field,
				c.Table.Type.Name(), c.Table.Name,
				rt.Type.Name(), rt.Name)
		}

		if c.NameLeft == "" {
			if columner != nil {
				c.NameLeft = columner.ColumnName(c.Field,
					c.Table.Type.Name(), c.Table.PrimaryKey.Field, c.Table.Name, c.Table.PrimaryKey.Name)
			} else {
				c.NameLeft = format.ColumnName(c.Field, c.Table.Type.Name(), c.Table.Name,
					c.Table.Type.Name(), c.Table.PrimaryKey.Field, c.Table.Name, c.Table.PrimaryKey.Name)
			}
		}

		if c.NameRight == "" {
			if columner != nil {
				c.NameRight = columner.ColumnName(c.Field,
					rt.Type.Name(), rt.PrimaryKey.Field, rt.Name, rt.PrimaryKey.Name)
			} else {
				c.NameRight = format.ColumnName(c.Field, c.Table.Type.Name(), c.Table.Name,
					rt.Type.Name(), rt.PrimaryKey.Field, rt.Name, rt.PrimaryKey.Name)
			}
		}

		if througher != nil {
			if ti, lf, rf := througher.ThroughTable(c.Field); ti != nil {
				tt, err := newTable(ti, x)
				if err != nil {
					return err
				}

				lc := tt.FindField(lf)
				if lc == nil {
					return fmt.Errorf("table: through field not found: %s/%s", tt.Type.Name(), lf)
				} else if lc.Relation != ManyToOne || lc.RelationTable.Type != c.Table.Type {
					return errors.New("table: through not correct: " + lc.FullName())
				}

				rc := tt.FindField(rf)
				if rc == nil {
					return fmt.Errorf("table: through field not found: %s/%s", tt.Type.Name(), rf)
				} else if rc.Relation != ManyToOne || rc.RelationTable.Type != c.RelationTable.Type {
					return errors.New("table: through not correct: " + rc.FullName())
				}

				c.ThroughTable = tt
				c.ThroughLeft = lc
				c.ThroughRight = rc
			}
		}

		if (c.ThroughTable == nil && c.NameLeft == c.NameRight) ||
			(c.ThroughTable != nil && c.ThroughLeft.Name == c.ThroughRight.Name) {
			return errors.New("table: many_to_many column name repeat: " + c.FullName())
		}
	}

	return nil
}

func (c *Column) PrimaryKey() bool {
	return c.Table.PrimaryKey != nil && c.Table.PrimaryKey.Index == c.Index
}

func (c *Column) AutoIncrement() bool {
	return c.Table.AutoIncrement != nil && c.Table.AutoIncrement.Index == c.Index
}

func (c *Column) AutoNowAdd() bool {
	return c.Table.AutoNowAdd != nil && c.Table.AutoNowAdd.Index == c.Index
}

func (c *Column) AutoNow() bool {
	return c.Table.AutoNow != nil && c.Table.AutoNow.Index == c.Index
}

func (c *Column) IsOneRelation() bool {
	return isOneRelation(c.Relation)
}

func (c *Column) IsManyRelation() bool {
	return isManyRelation(c.Relation)
}

func (c *Column) HasGetter() bool {
	return c.Getter != -1
}

func (c *Column) HasSetter() bool {
	return c.Setter != -1
}

func (c *Column) FullName() string {
	return c.Table.Type.Name() + "." + c.Field
}

// many relation get the field value
func (c *Column) GetValue(v reflect.Value) (interface{}, error) {
	if t := v.Type(); t != c.Table.Type {
		return nil, errors.New("table: get value type mismatching: " + c.FullName())
	}

	if isOneRelation(c.Relation) {
		v = v.Field(c.Index)
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				return nil, nil
			}
			v = v.Elem()
		}
		return c.RelationTable.PrimaryKey.GetValue(v)
	}

	if c.HasGetter() {
		var m reflect.Value
		if c.GetPointer {
			if !v.CanAddr() {
				return nil, errors.New("table: getter need pointer receiver: " + c.FullName())
			}
			m = v.Addr().Method(c.Getter)
		} else {
			m = v.Method(c.Getter)
		}
		if out := m.Call(nil); c.GetError && !out[1].IsNil() {
			return nil, out[1].Interface().(error)
		} else {
			return out[0].Interface(), nil
		}
	}

	return v.Field(c.Index).Interface(), nil
}

// many relation set the field value
func (c *Column) SetValue(v reflect.Value, i interface{}) error {
	if t := v.Type(); t != c.Table.Type {
		return errors.New("table: set value type mismatching: " + c.FullName())
	}

	if isOneRelation(c.Relation) {
		v = v.Field(c.Index)
		if v.Kind() == reflect.Ptr {
			if v.IsNil() {
				if !v.CanSet() {
					goto failed
				}
				v.Set(reflect.New(c.RelationTable.Type))
			}
			v = v.Elem()
		}
		return c.RelationTable.PrimaryKey.SetValue(v, i)
	}

	if c.HasSetter() {
		if !v.CanAddr() {
			return errors.New("table: setter need pointer receiver: " + c.FullName())
		}
		in := make([]reflect.Value, 1)
		if c.SetType == typeInterface {
			in[0] = reflect.ValueOf(i)
		} else if c.SetType == TypeByteSlice {
			if i != nil {
				if x, ok := i.([]byte); ok {
					in[0] = reflect.ValueOf(x)
				} else if x, ok := i.(string); ok {
					in[0] = reflect.ValueOf([]byte(x))
				} else {
					goto failed
				}
			} else {
				in[0] = reflect.ValueOf(i)
			}
		} else if c.SetType == TypeTime {
			if x, ok := i.(time.Time); ok {
				in[0] = reflect.ValueOf(x)
			} else {
				goto failed
			}
		} else {
			switch c.SetType.Kind() {
			case reflect.Bool:
				if x, ok := i.(bool); ok {
					in[0] = reflect.ValueOf(x)
				} else {
					goto failed
				}
			case reflect.Int64:
				if x, ok := i.(int64); ok {
					in[0] = reflect.ValueOf(x)
				} else {
					goto failed
				}
			case reflect.Float64:
				if x, ok := i.(float64); ok {
					in[0] = reflect.ValueOf(x)
				} else {
					goto failed
				}
			case reflect.String:
				if x, ok := i.(string); ok {
					in[0] = reflect.ValueOf(x)
				} else if x, ok := i.([]byte); ok {
					in[0] = reflect.ValueOf(string(x))
				} else {
					goto failed
				}
			default:
				goto failed
			}
		}
		if out := v.Addr().Method(c.Setter).Call(in); c.SetError && !out[0].IsNil() {
			return out[0].Interface().(error)
		}
		return nil
	}

	v = v.Field(c.Index)
	if v.CanSet() && (func() (ok bool) {
		defer func() {
			if recover() != nil {
				ok = false
			}
		}()
		if c.AutoIncrement() {
			var ai int64
			switch j := i.(type) {
			case int64:
				ai = j
			case int:
				ai = int64(j)
			case int8:
				ai = int64(j)
			case int16:
				ai = int64(j)
			case int32:
				ai = int64(j)
			case uint:
				ai = int64(j)
			case uint8:
				ai = int64(j)
			case uint16:
				ai = int64(j)
			case uint32:
				ai = int64(j)
			case uint64:
				ai = int64(j)
			default:
				return false
			}
			switch autoIncrement(v.Kind()) {
			case -1:
				v.SetInt(ai)
			case 1:
				v.SetUint(uint64(ai))
			default:
				return false
			}
		} else {
			v.Set(reflect.ValueOf(i))
		}
		return true
	}()) {
		return nil
	}

failed:
	return errors.New("table: set value failed: " + c.FullName())
}

// panic if v is not table's type
func (c *Column) Scan(v reflect.Value) interface{} {
	if v.Type() != c.Table.Type {
		panic("table: scan type mismatching: " + c.FullName())
	}

	if c.HasSetter() {
		return reflect.New(c.SetType).Interface()
	} else {
		fv := v.Field(c.Index)
		if c.IsOneRelation() {
			rt := c.RelationTable
			for {
				if fv.Kind() == reflect.Ptr {
					if fv.IsNil() {
						fv.Set(reflect.New(rt.Type))
					}
					fv = fv.Elem()
				}
				fv = fv.Field(rt.PrimaryKey.Index)
				if rt.PrimaryKey.IsOneRelation() {
					rt = rt.PrimaryKey.RelationTable
				} else {
					break
				}
			}
		}
		if fv.Kind() == reflect.Ptr { // cxr? *int
			if fv.IsNil() {
				fv.Set(reflect.New(fv.Type().Elem()))
			}
			return fv.Interface()
		} else {
			return fv.Addr().Interface()
		}
	}
}

var (
	tmutex = new(sync.RWMutex)
	tables = make(map[reflect.Type]*Table)
)

func NewTable(i interface{}) (*Table, error) {
	return newTable(i, nil)
}

func newTable(i interface{}, x map[reflect.Type]*Table) (*Table, error) {
	if table, ok := i.(*Table); ok {
		return table, nil
	}

	t := reflect.TypeOf(i)
	k := t.Kind()
	if k == reflect.Ptr {
		t = t.Elem()
		k = t.Kind()
	}
	if k == reflect.Array || k == reflect.Slice {
		t = t.Elem()
		k = t.Kind()
		if k == reflect.Ptr {
			t = t.Elem()
			k = t.Kind()
		}
	}

	if x == nil {
		x = make(map[reflect.Type]*Table)
	}

	return tableOf(t, x)
}

func TableOf(t reflect.Type) (*Table, error) {
	return tableOf(t, make(map[reflect.Type]*Table))
}

func tableOf(t reflect.Type, x map[reflect.Type]*Table) (*Table, error) {
	if t.Kind() != reflect.Struct {
		return nil, errors.New("table not struct")
	}

	tmutex.RLock()
	table, ok := tables[t]
	tmutex.RUnlock()
	if ok {
		return table, nil
	}

	table, ok = x[t]
	if ok {
		return table, nil
	}

	table = &Table{
		Type:      t,
		Value:     reflect.New(t),
		Name:      t.Name(),
		FieldMap:  make(map[string]*Column),
		ColumnMap: make(map[string]*Column),
	}

	x[t] = table

	if i, ok := table.Value.Interface().(Tabler); ok {
		table.Name = i.TableName()
	}

	for i, n := 0, t.NumField(); i < n; i++ {
		f := t.Field(i)
		if f.PkgPath != "" {
			continue
		}

		tag := f.Tag.Get("scrud")
		if tag == "" && strings.Index(string(f.Tag), ":") < 0 {
			tag = string(f.Tag)
		}
		if tag == "-" {
			continue
		}

		c := &Column{
			Table:  table,
			Type:   f.Type,
			Index:  i,
			Field:  f.Name,
			Getter: -1,
			Setter: -1,
		}

		table.FieldMap[f.Name] = c

		if a := strings.Split(tag, ","); len(a) > 1 {
			tag = a[0]
			for _, o := range a[1:] {
				switch o {
				case "primary_key":
					if table.PrimaryKey != nil {
						return nil, errors.New("table: more than one primary_key: " + t.Name())
					}
					table.PrimaryKey = c
				case "auto_increment":
					if table.AutoIncrement != nil {
						return nil, errors.New("table: more than one auto_increment: " + t.Name())
					}
					if autoIncrement(f.Type.Kind()) != 0 {
						table.AutoIncrement = c
					} else {
						return nil, errors.New("table: auto_increment not ints or uints: " + c.FullName())
					}
				case "auto_now_add":
					if table.AutoNowAdd != nil {
						return nil, errors.New("table: more than one auto_now_add: " + t.Name())
					}
					if f.Type != TypeTime {
						return nil, errors.New("table: auto_now_add not time.Time: " + c.FullName())
					}
					if table.AutoNow != nil && table.AutoNow.Index == c.Index {
						return nil, errors.New("table: auto_now_add and auto_now both appear: " + c.FullName())
					}
					table.AutoNowAdd = c
				case "auto_now":
					if table.AutoNow != nil {
						return nil, errors.New("table: more than one auto_now: " + t.Name())
					}
					if f.Type != TypeTime {
						return nil, errors.New("table: auto_now not time.Time: " + c.FullName())
					}
					if table.AutoNowAdd != nil && table.AutoNowAdd.Index == c.Index {
						return nil, errors.New("table: auto_now and auto_now_add both appear: " + c.FullName())
					}
					table.AutoNow = c
				default:
					if r, ok := relations[o]; ok {
						if c.Relation != 0 {
							return nil, errors.New("table: more than one relation: " + c.FullName())
						}

						ft := f.Type
						fk := ft.Kind()
						if fk == reflect.Ptr {
							ft = ft.Elem()
							fk = ft.Kind()
						}

						if isManyRelation(r) {
							if fk != reflect.Slice {
								return nil, errors.New("table: many relation not slice: " + c.FullName())
							}
							ft = ft.Elem()
							fk = ft.Kind()
							if fk == reflect.Ptr {
								ft = ft.Elem()
								fk = ft.Kind()
							}
						}

						rt, err := tableOf(ft, x)
						if err != nil {
							return nil, err
						}
						c.Relation = r
						c.RelationTable = rt
					} else {
						return nil, fmt.Errorf("table: unknown option: %s/%s", c.FullName(), o)
					}
				}
			}
		}

		c.Name = tag

		c.Valuer = f.Type.Implements(typeValuer)
		c.Scanner = f.Type.Implements(typeScanner)

		mn := "ScrudGet" + f.Name
		if m, ok := table.Value.Type().MethodByName(mn); ok {
			_, has := t.MethodByName(mn)
			if o := m.Type.NumOut(); m.Type.NumIn() == 1 &&
				((o == 1 && isValueType(m.Type.Out(0))) ||
					(o == 2 && isValueType(m.Type.Out(0)) && m.Type.Out(1) == typeError)) {
				c.Getter = m.Index
				c.GetError = o == 2
				c.GetType = m.Type.Out(0)
				c.GetPointer = !has
			} else {
				return nil, errors.New("table: getter not correct: " + c.FullName())
			}
		}

		mn = "ScrudSet" + f.Name
		if m, ok := table.Value.Type().MethodByName(mn); ok {
			_, has := t.MethodByName(mn)
			if o := m.Type.NumOut(); !has &&
				m.Type.NumIn() == 2 && isValueType(m.Type.In(1)) &&
				(o == 0 || (o == 1 && m.Type.Out(0) == typeError)) {
				c.Setter = m.Index
				c.SetError = o == 1
				c.SetType = m.Type.In(1)
			} else {
				return nil, errors.New("table: setter not correct: " + c.FullName())
			}
		}

		table.Columns = append(table.Columns, c)
	}

	if len(table.Columns) == 0 {
		return nil, errors.New("table: no columns: " + t.Name())
	}

	if table.PrimaryKey == nil {
		if table.AutoIncrement == nil {
			if c, ok := table.FieldMap["Id"]; ok {
				if autoIncrement(t.Field(c.Index).Type.Kind()) != 0 {
					table.AutoIncrement = c
				}
			}
		}
		if table.AutoIncrement != nil {
			table.PrimaryKey = table.AutoIncrement
		}
	}

	if table.PrimaryKey != nil && isManyRelation(table.PrimaryKey.Relation) {
		return nil, errors.New("table: many relation on primary_key: " + t.Name())
	}

	if table.AutoIncrement != nil &&
		(table.AutoIncrement.HasGetter() || table.AutoIncrement.HasSetter()) {
		return nil, errors.New("table: auto_increment not allow getter and setter: " + table.AutoIncrement.FullName())
	}

	if table.AutoNowAdd != nil &&
		(table.AutoNowAdd.HasGetter() || table.AutoNowAdd.HasSetter()) {
		return nil, errors.New("table: auto_now_add not allow getter and setter: " + table.AutoNowAdd.FullName())
	}

	if table.AutoNow != nil &&
		(table.AutoNow.HasGetter() || table.AutoNow.HasSetter()) {
		return nil, errors.New("table: auto_now not allow getter and setter: " + table.AutoNow.FullName())
	}

	if table.PrimaryKey != nil {
		if err := table.PrimaryKey.init(x); err != nil {
			return nil, err
		}
	}
	for _, c := range table.Columns {
		if !c.PrimaryKey() {
			if err := c.init(x); err != nil {
				return nil, err
			}
		}
	}

	tmutex.Lock()
	tables[t] = table
	tmutex.Unlock()

	return table, nil
}

const (
	OneToOne = iota + 1
	OneToMany
	ManyToOne
	ManyToMany
	ForeignKey = ManyToOne
)

var relations = map[string]int{
	"one_to_one":   OneToOne,
	"one_to_many":  OneToMany,
	"many_to_one":  ManyToOne,
	"foreign_key":  ForeignKey,
	"many_to_many": ManyToMany,
}

func isOneRelation(r int) bool {
	return r == OneToOne || r == ManyToOne
}

func isManyRelation(r int) bool {
	return r == OneToMany || r == ManyToMany
}

var (
	typeValuer    = reflect.TypeOf((*driver.Valuer)(nil)).Elem()
	typeScanner   = reflect.TypeOf((*sql.Scanner)(nil)).Elem()
	typeError     = reflect.TypeOf((*error)(nil)).Elem()
	typeInterface = reflect.TypeOf((*interface{})(nil)).Elem()
	TypeString    = reflect.TypeOf("")
	TypeByteSlice = reflect.TypeOf(([]byte)(nil))
	TypeTime      = reflect.TypeOf(time.Time{})
)

func autoIncrement(k reflect.Kind) int8 {
	switch k {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return -1
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return 1
	}
	return 0
}

func isValueType(t reflect.Type) bool {
	if t == typeInterface || t == TypeByteSlice || t == TypeTime {
		return true
	}
	switch t.Kind() {
	case reflect.Bool, reflect.Int64, reflect.Float64, reflect.String:
		return true
	}
	return false
}
