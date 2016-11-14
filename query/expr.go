// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import (
	"errors"
	"fmt"
	"strings"
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
			if c := v.(string); strings.Contains(c, ".") {
				runes := []rune(c)
				n := len(runes)
				d, e := make([]rune, 0, 2), make([]string, 0, 2)
				f := func() {
					e = append(e, s.FormatName(string(d)))
					d = d[0:0]
				}
				for i := 0; i < n; i++ {
					if r := runes[i]; r == '.' {
						if j := i + 1; j < n && runes[j] == r {
							i = j
							d = append(d, r)
						} else if i < n-1 {
							f()
						}
					} else {
						d = append(d, r)
					}
				}
				f()
				a = append(a, strings.Join(e, "."))
			} else {
				a = append(a, s.FormatName(c))
			}
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
		return errExpr("empty expression")
	}

	runes, types := []rune(format), make([]int, 0, 2)
	n := len(runes)
	a := make([]rune, 0, n)
	count, index := len(args), 0
	b := make([]interface{}, 0, count)

	for i := 0; i < n; i++ {
		if r := runes[i]; r == '`' {
			if j := i + 1; j < n && runes[j] == r {
				i = j
				a = append(a, r)
			} else {
				c := make([]rune, 0)
				for ; j < n; j++ {
					if r := runes[j]; r == '`' {
						if k := j + 1; k < n && runes[k] == r {
							j = k
							c = append(c, r)
						} else {
							break
						}
					} else {
						c = append(c, r)
					}
				}
				if j < n {
					i = j
					a = append(a, '%', 's')
					b = append(b, string(c))
					types = append(types, typeIdentifier)
				} else {
					return errExpr("expression back quote not double: " + format)
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
					return errExpr("expression args not enough: " + format)
				}
			}
		} else if r == '%' {
			a = append(a, r, r)
		} else {
			a = append(a, r)
		}
	}

	if index < count {
		return errExpr("expression args too many: " + format)
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
