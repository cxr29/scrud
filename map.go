package scrud

import (
	"errors"
	"reflect"

	"github.com/cxr29/scrud/internal/table"
)

// return map's key is column name
func StructToMap(data interface{}, columns ...string) (map[string]interface{}, error) {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() == reflect.Ptr {
		if v.IsNil() {
			return nil, errors.New("scrud: s2m nil")
		}
		v = v.Elem()
		t = v.Type()
	}

	x, err := table.TableOf(t)
	if err != nil {
		return nil, err
	}

	columnMap, exclude, err := tidyColumns("s2m", x, columns...)
	if err != nil {
		return nil, err
	}
	count := len(columnMap)

	m := make(map[string]interface{}, len(x.Columns))

	for _, c := range x.Columns {
		if c.IsManyRelation() {
			continue
		}
		if count > 0 {
			if _, ok := columnMap[c.Index]; (ok && exclude) || (!ok && !exclude) {
				continue
			}
		}
		if i, err := c.GetValue(v); err != nil {
			return nil, err
		} else {
			m[c.Name] = i
		}
	}

	return m, nil
}

// map's key should be column name
func MapToStruct(data interface{}, m map[string]interface{}) error {
	v := reflect.ValueOf(data)
	t := v.Type()
	if v.Kind() != reflect.Ptr {
		return errors.New("scrud: m2s need pointer")
	}
	if v.IsNil() {
		return errors.New("scrud: m2s nil")
	}
	v = v.Elem()
	t = v.Type()

	x, err := table.TableOf(t)
	if err != nil {
		return err
	}

	for _, c := range x.Columns {
		if c.IsManyRelation() {
			continue
		}

		if i, ok := m[c.Name]; !ok {
			continue
		} else if err := c.SetValue(v, i); err != nil {
			return err
		}
	}

	return nil
}
