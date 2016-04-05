// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import (
	"bytes"
	"errors"
	"strconv"
)

type delete struct {
	alias string
	table string
	where []Condition
	order []interface{}
	limit int
}

// delete clause generator
func Delete(table string) *delete {
	return &delete{table: table}
}

func (d *delete) Err() error {
	return nil
}

func (d *delete) Expand(s Starter) (string, []interface{}, error) {
	buf, args := new(bytes.Buffer), make([]interface{}, 0)

	buf.WriteString("DELETE FROM ")
	buf.WriteString(s.FormatName(d.table))

	if n := len(d.where); n > 0 {
		buf.WriteString(" WHERE ")
		e, a, err := And(d.where...).Expand(s)
		if err != nil {
			return "", nil, err
		}
		buf.WriteString(e)
		args = append(args, a...)
	}

	if n := len(d.order); n > 0 {
		buf.WriteString(" ORDER BY ")
		for k, v := range d.order {
			switch i := v.(type) {
			case string:
				buf.WriteString(s.FormatName(i))
			case Expression:
				e, a, err := i.Expand(s)
				if err != nil {
					return "", nil, err
				}
				buf.WriteString(e)
				args = append(args, a...)
			default:
				return "", nil, errors.New("delete: order by must be string or expression")
			}
			if k < n-1 {
				buf.WriteByte(',')
			}
		}
	}

	if d.limit > 0 {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.Itoa(d.limit))
	}

	return buf.String(), args, nil
}

func (d *delete) Where(a ...Condition) *delete {
	d.where = append(d.where, a...)
	return d
}

func (d *delete) OrderBy(a ...interface{}) *delete {
	d.order = append(d.order, a...)
	return d
}

func (d *delete) Limit(n int) *delete {
	d.limit = n
	return d
}
