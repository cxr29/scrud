// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import (
	"bytes"
	"errors"
)

type create struct {
	alias   string
	table   string
	columns []string
	values  [][]interface{}
}

// insert clause generator
func Insert(table string) *create {
	return &create{table: table}
}

func (c *create) Err() error {
	return nil
}

func (c *create) Expand(s Starter) (string, []interface{}, error) {
	x, y := len(c.columns), len(c.values)
	if x == 0 {
		return "", nil, errors.New("create: no columns")
	}
	if y == 0 {
		return "", nil, errors.New("create: no values")
	}

	buf, args := new(bytes.Buffer), make([]interface{}, 0, x*y)

	buf.WriteString("INSERT INTO ")
	buf.WriteString(s.FormatName(c.table))
	buf.WriteString(" (")

	for k, v := range c.columns {
		buf.WriteString(s.FormatName(v))
		if k < x-1 {
			buf.WriteByte(',')
		}
	}

	buf.WriteString(") VALUES ")

	for k, v := range c.values {
		if len(v) != x {
			return "", nil, errors.New("create: columns count not equal values count")
		}
		buf.WriteByte('(')
		for i := 0; i < x; i++ {
			buf.WriteString(s.NextMarker())
			if i < x-1 {
				buf.WriteByte(',')
			}
		}
		buf.WriteByte(')')
		if k < y-1 {
			buf.WriteByte(',')
		}
		args = append(args, v...)
	}

	return buf.String(), args, nil
}

func (c *create) Alias() string {
	return c.alias
}

func (c *create) As(s string) Querier {
	c.alias = s
	return c
}

func (c *create) Columns(a ...string) *create {
	c.columns = append(c.columns, a...)
	return c
}

func (c *create) Values(a ...interface{}) *create {
	c.values = append(c.values, a)
	return c
}
