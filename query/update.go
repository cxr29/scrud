// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import (
	"bytes"
	"errors"
	"strconv"
	"time"
)

type set struct {
	key   string
	value interface{}
}

type update struct {
	alias string
	table string
	set   []set
	where []Condition
	order []interface{}
	limit int
}

// update clause generator
func Update(table string) *update {
	return &update{table: table}
}

func (u *update) Err() error {
	return nil
}

func (u *update) Expand(s Starter) (string, []interface{}, error) {
	buf, args := new(bytes.Buffer), make([]interface{}, 0)

	buf.WriteString("UPDATE ")
	buf.WriteString(s.FormatName(u.table))

	if n := len(u.set); n > 0 {
		buf.WriteString(" SET ")
		for k, v := range u.set {
			buf.WriteString(s.FormatName(v.key))
			buf.WriteByte('=')
			if i, ok := v.value.(Expression); ok {
				e, a, err := i.Expand(s)
				if err != nil {
					return "", nil, err
				}
				buf.WriteString(e)
				args = append(args, a...)
			} else {
				buf.WriteString(s.NextMarker())
				args = append(args, v.value)
			}
			if k < n-1 {
				buf.WriteByte(',')
			}
		}
	} else {
		return "", nil, errors.New("update: empty set")
	}

	if n := len(u.where); n > 0 {
		buf.WriteString(" WHERE ")
		e, a, err := And(u.where...).Expand(s)
		if err != nil {
			return "", nil, err
		}
		buf.WriteString(e)
		args = append(args, a...)
	}

	if n := len(u.order); n > 0 {
		buf.WriteString(" ORDER BY ")
		for k, v := range u.order {
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
				return "", nil, errors.New("update: order by must be string or expression")
			}
			if k < n-1 {
				buf.WriteByte(',')
			}
		}
	}

	if u.limit > 0 {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.Itoa(u.limit))
	}

	return buf.String(), args, nil
}

func (u *update) Alias() string {
	return u.alias
}

func (u *update) As(s string) Querier {
	u.alias = s
	return u
}

func (u *update) Set(k string, v interface{}) *update {
	u.set = append(u.set, set{k, v})
	return u
}

func (u *update) AutoNow(k string) *update {
	u.set = append(u.set, set{k, time.Unix(time.Now().Unix(), 0)})
	return u
}

func (u *update) Where(a ...Condition) *update {
	u.where = append(u.where, a...)
	return u
}

func (u *update) OrderBy(a ...interface{}) *update {
	u.order = append(u.order, a...)
	return u
}

func (u *update) Limit(n int) *update {
	u.limit = n
	return u
}
