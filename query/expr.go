// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import (
	"errors"
	"fmt"
)

const (
	typeMarker = iota + 1
	typeIdentifier
	typeExpression
)

type expr struct {
	err    error
	format string
	types  []int
	args   []interface{}
}

func (e *expr) Err() error {
	return e.err
}

func (e *expr) Expand(s Starter) (string, []interface{}, error) {
	if e.err != nil {
		return "", nil, e.err
	}

	a := make([]interface{}, 0)
	b := make([]interface{}, 0)

	for k, v := range e.args {
		switch e.types[k] {
		case typeMarker:
			a = append(a, s.NextMarker())
			b = append(b, v)
		case typeIdentifier:
			c := v.([2]string)
			d := s.FormatName(c[1])
			if c[0] != "" {
				d = s.FormatName(c[0]) + "." + d
			}
			a = append(a, d)
		case typeExpression:
			c, d, err := v.(Expression).Expand(s)
			if err != nil {
				return "", nil, err
			}
			a = append(a, c)
			b = append(b, d...)
		}
	}

	return fmt.Sprintf(e.format, a...), b, nil
}

func errExpr(s string) *expr {
	return &expr{err: errors.New(s)}
}

func newExpr(format string, args ...interface{}) *expr {
	if format == "" {
		return errExpr("expression: empty format")
	}

	runes, types := []rune(format), make([]int, 0)
	a, b := make([]rune, 0), make([]interface{}, 0)
	count, index, n := len(args), 0, len(runes)

	for i := 0; i < n; i++ {
		if r := runes[i]; r == '`' {
			if j := i + 1; j < n && runes[j] == r {
				i = j
				a = append(a, r)
			} else {
				c := make([]rune, 0)
				d := -1
				for ; j < n; j++ {
					if r := runes[j]; r == '`' {
						if k := j + 1; k < n && runes[k] == r {
							j = k
							c = append(c, r)
						} else {
							break
						}
					} else if r == '.' {
						if k := j + 1; k < n && runes[k] == r {
							j = k
							c = append(c, r)
						} else if d == -1 {
							d = len(c)
						} else {
							return errExpr("expression: dot not double or too many: " + format)
						}
						/*
							} else if r == '?' {
								if k := j + 1; k < n && runes[k] == r {
									j = k
									c = append(c, r)
								} else {
									return errExpr("expression: marker not double: " + format)
								}
						*/
					} else {
						c = append(c, r)
					}
				}
				if j < n {
					i = j
					a = append(a, '%', 's')
					var v [2]string
					if d == -1 {
						v[1] = string(c)
					} else {
						v[0], v[1] = string(c[:d]), string(c[d:])
					}
					b = append(b, v)
					types = append(types, typeIdentifier)
				} else {
					return errExpr("expression: back quote not double or not close: " + format)
				}
			}
		} else if r == '?' {
			if j := i + 1; j < n && runes[j] == r {
				i = j
				a = append(a, '?')
			} else {
				a = append(a, '%', 's')
				if index < count {
					v := args[index]
					index++
					b = append(b, v)
					if _, ok := v.(Expression); ok {
						types = append(types, typeExpression)
					} else {
						types = append(types, typeMarker)
					}
				} else {
					return errExpr("expression: argument not enough: " + format)
				}
			}
		} else if r == '%' {
			a = append(a, r, r)
		} else {
			a = append(a, r)
		}
	}

	if index < count {
		return errExpr("expression: too many argument: " + format)
	}

	return &expr{format: string(a), types: types, args: b}
}

// marker: question mark(?) as placeholder, recursive expand if corresponding argument is expression
//
// identifier: the name in back quote(`) such as `column` and `table.column`
//
// double back quote(`) and dot sign(.) to include it as literal in identifier
//
// double back quote(`) and question mark(?) to include it as literal in expression
func Expr(format string, args ...interface{}) Expression {
	return newExpr(format, args...)
}
