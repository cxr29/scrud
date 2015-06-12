package scrud

import (
	"database/sql"
	"errors"
	"fmt"
	"reflect"

	"github.com/cxr29/scrud/internal/table"
)

var ErrNoRows = sql.ErrNoRows

type Rows struct {
	err error
	*sql.Rows
	cnt  int
	cols []string
}

func (r *Rows) Err() error {
	if r.err != nil {
		return r.err
	}

	return r.Rows.Err()
}

func (r *Rows) columns(x *table.Table) ([]*table.Column, error) {
	m := make(map[int]struct{}, r.cnt)
	cols := make([]*table.Column, r.cnt)
	for k, v := range r.cols {
		c := x.FindColumn(v)
		if c == nil {
			return nil, fmt.Errorf("scrud: scan column not found: %s/%s", x.Type.Name(), v)
		}
		if c.IsManyRelation() {
			return nil, errors.New("scrud: scan many relation column: " + c.FullName())
		}
		if _, ok := m[c.Index]; ok {
			return nil, errors.New("scrud: scan column repeat: " + c.FullName())
		} else {
			m[c.Index] = struct{}{}
		}
		cols[k] = c
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
	scans := make([]interface{}, len(cols))
	for i, c := range cols {
		scans[i] = c.Scan(v)
		if c.HasEncoding() || c.HasSetter() {
			set[i] = c
		}
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

func (r *Rows) types(i interface{}) ([]reflect.Type, map[int]*table.Column, error) {
	a := make([]reflect.Type, r.cnt)
	b := make(map[int]*table.Column)
	m := make(map[string]struct{}, r.cnt)
	switch x := i.(type) {
	case nil, string:
		for j := range a {
			a[j] = table.TypeString
		}
	case []interface{}:
		if len(x) != r.cnt {
			return nil, nil, errors.New("scrud: map scan slice length")
		}

		for k, v := range r.cols {
			if _, ok := m[v]; ok {
				return nil, nil, errors.New("scrud: map scan column repeat: " + v)
			} else {
				m[v] = struct{}{}
			}

			a[k] = reflect.TypeOf(x[k])
		}
	case map[string]interface{}:
		for k, v := range r.cols {
			if _, ok := m[v]; ok {
				return nil, nil, errors.New("scrud: map scan column repeat: " + v)
			} else {
				m[v] = struct{}{}
			}

			if j, ok := x[v]; ok {
				a[k] = reflect.TypeOf(j)
			} else {
				a[k] = table.TypeString
			}
		}
	default:
		t := reflect.TypeOf(i)
		if t.Implements(table.TypeScanner) {
			for j := range a {
				a[j] = t
			}
		} else {
			switch t.Kind() {
			case reflect.Bool,
				reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64,
				reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64,
				reflect.Float32, reflect.Float64:
				for j := range a {
					a[j] = t
				}
			default:
				switch t {
				case table.TypeTime, table.TypeByteSlice:
					for j := range a {
						a[j] = t
					}
				default:
					x, err := table.NewTable(i)
					if err != nil {
						return nil, nil, err
					}
					for k, v := range r.cols {
						if _, ok := m[v]; ok {
							return nil, nil, errors.New("scrud: map scan column repeat: " + v)
						} else {
							m[v] = struct{}{}
						}

						if c := x.FindColumn(v); c != nil {
							if c.IsManyRelation() {
								return nil, nil, errors.New("scrud: map scan many relation column: " + c.FullName())
							}
							for c.IsOneRelation() {
								c = c.RelationTable.PrimaryKey
							}
							if c.HasEncoding() {
								a[k] = table.TypeByteSlice
								b[k] = c
							} else if c.HasSetter() {
								a[k] = c.SetType
							} else {
								a[k] = c.Type
							}
						} else {
							a[k] = table.TypeString
						}
					}
				}
			}
		}
	}
	return a, b, nil
}

func (r *Rows) mapScan(a []reflect.Type, b map[int]*table.Column) (map[string]interface{}, error) {
	scans := make([]interface{}, r.cnt)
	for k, v := range a {
		if v.Kind() == reflect.Ptr {
			v = v.Elem()
		}
		scans[k] = reflect.New(v).Interface()
	}

	if err := r.Rows.Scan(scans...); err != nil {
		return nil, err
	}

	data := make(map[string]interface{}, r.cnt)
	for k, v := range scans {
		if a[k].Kind() == reflect.Ptr {
			data[r.cols[k]] = v
		} else {
			i := reflect.ValueOf(v).Elem().Interface()
			if c, ok := b[k]; ok {
				var err error
				i, err = c.Decode(i.([]byte))
				if err != nil {
					return nil, err
				}
			}
			data[r.cols[k]] = i
		}
	}
	return data, nil
}

// i: column type, nil/bool/ints/uints/floats/string/time.Time/[]byte/[]interface{}/map[string]interface{}/sql.Scanner/struct
//
// if struct, setter column will be the setter type and not call the setter
func (r *Rows) MapScan(i interface{}) (map[string]interface{}, error) {
	if r.err != nil {
		return nil, r.err
	}

	a, b, err := r.types(i)
	if err != nil {
		return nil, err
	}
	return r.mapScan(a, b)
}

// scan one row as map then close the rows
//
// i: column type, nil/bool/ints/uints/floats/string/time.Time/[]byte/[]interface{}/map[string]interface{}/sql.Scanner/struct
//
// if struct, setter column will be the setter type and not call the setter
func (r *Rows) MapOne(i interface{}) (map[string]interface{}, error) {
	if r.err != nil {
		return nil, r.err
	}

	defer r.Close()

	if r.Next() {
		return r.MapScan(i)
	} else {
		return nil, ErrNoRows
	}
}

// scan rows as slice of map then close the rows
//
// i: column type, nil/bool/ints/uints/floats/string/time.Time/[]byte/[]interface{}/map[string]interface{}/sql.Scanner/struct
//
// if struct, setter column will be the setter type and not call the setter
func (r *Rows) MapAll(i interface{}) ([]map[string]interface{}, error) {
	if r.err != nil {
		return nil, r.err
	}

	defer r.Close()

	a, b, err := r.types(i)
	if err != nil {
		return nil, err
	}

	data := make([]map[string]interface{}, 0)

	for r.Next() {
		if m, err := r.mapScan(a, b); err != nil {
			return nil, err
		} else {
			data = append(data, m)
		}
	}
	if err := r.Err(); err != nil {
		return nil, err
	}

	return data, nil
}

func (r *Rows) Row(dest ...interface{}) error {
	if r.err != nil {
		return r.err
	}

	defer r.Close()

	if r.Next() {
		return r.Rows.Scan(dest...)
	} else {
		return ErrNoRows
	}
}
