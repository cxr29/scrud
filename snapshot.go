package scrud

import (
	"errors"
	"reflect"
	"time"

	"github.com/cxr29/scrud/format"
	"github.com/cxr29/scrud/internal/table"
	. "github.com/cxr29/scrud/query"
)

func (db *DB) Snapshot() *Snapshot {
	return &Snapshot{xr: db}
}

func (tx *Tx) Snapshot() *Snapshot {
	return &Snapshot{xr: tx}
}

// snapshot manager
type Snapshot struct {
	xr faker
}

func (s *Snapshot) Insert(data interface{}) (int64, time.Time, error) {
	var zt time.Time

	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return 0, zt, errors.New("scrud: snapshot insert nil")
		}
		v = v.Elem()
		t = v.Type()
	}

	x, err := table.TableOf(t)
	if err != nil {
		return 0, zt, err
	}

	tableName, _, timeName := format.SnapshotName(x.Type.Name(), x.Name)

	i := Insert(tableName)

	st := time.Now()
	i.Columns(timeName)
	values := []interface{}{st}

	for _, c := range x.Columns {
		if c.IsManyRelation() {
			continue
		}
		if v, err := c.GetValue(v); err != nil {
			return 0, zt, err
		} else {
			i.Columns(c.Name)
			values = append(values, v)
		}
	}
	i.Values(values)

	q, a, err := i.Expand(s.xr.Starter())
	if err != nil {
		return 0, zt, err
	}

	r, err := s.xr.Exec(q, a...)
	if err != nil {
		return 0, zt, err
	}

	ai, err := r.LastInsertId()
	if err != nil {
		return 0, zt, err
	}

	return ai, st, nil
}

func (s *Snapshot) Select(id int64, data interface{}, columns ...string) (time.Time, error) {
	var zt time.Time

	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return zt, errors.New("scrud: snapshot select nil")
		}
		v = v.Elem()
		t = v.Type()
	}

	x, err := table.TableOf(t)
	if err != nil {
		return zt, err
	}

	tableName, idName, timeName := format.SnapshotName(x.Type.Name(), x.Name)

	columnMap, exclude, err := tidyColumns("snapshot select", x, columns...)
	if err != nil {
		return zt, err
	}
	count := len(columnMap)

	var st time.Time

	elect := []interface{}{timeName}
	set := make(map[int]*table.Column)
	scans := []interface{}{&st}
	for _, c := range x.Columns {
		if c.IsManyRelation() {
			continue
		}
		if count > 0 {
			if _, ok := columnMap[c.Index]; (ok && exclude) || (!ok && !exclude) {
				continue
			}
		}
		elect = append(elect, c.Name)
		if c.HasEncoding() || c.HasSetter() {
			set[len(scans)] = c
		}
		scans = append(scans, c.Scan(v))
	}

	q, a, err := Select(elect...).From(tableName).Where(Eq(idName, id)).Expand(s.xr.Starter())
	if err != nil {
		return zt, err
	}

	if err := s.xr.QueryRow(q, a...).Scan(scans...); err != nil {
		return zt, err
	}

	for i, c := range set {
		if err := c.SetValue(v, reflect.ValueOf(scans[i]).Elem().Interface()); err != nil {
			return zt, err
		}
	}

	return st, nil
}

func (s *Snapshot) Delete(id int64, data interface{}) error {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return errors.New("scrud: snapshot delete nil")
		}
		v = v.Elem()
		t = v.Type()
	}

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	tableName, idName, _ := format.SnapshotName(x.Type.Name(), x.Name)

	q, a, err := Delete(tableName).Where(Eq(idName, id)).Limit(1).Expand(s.xr.Starter())
	if err == nil {
		_, err = s.xr.Exec(q, a...)
	}

	return err
}
