// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import "strings"

type cond struct {
	not  bool
	expr *expr
}

func (c *cond) Err() error {
	return c.expr.err
}

func (c *cond) Expand(s Starter) (string, []interface{}, error) {
	q, a, err := c.expr.Expand(s)
	if err != nil {
		return "", nil, err
	}
	if c.not {
		q = "NOT (" + q + ")"
	}
	return q, a, nil
}

func (c *cond) Not() Condition {
	return &cond{not: !c.not, expr: c.expr}
}

func (c *cond) And(a ...Condition) Condition {
	return And(append([]Condition{c}, a...)...)
}

func (c *cond) Or(a ...Condition) Condition {
	return Or(append([]Condition{c}, a...)...)
}

type logic struct {
	err  error // first error
	and  bool
	cond []Condition
}

func (l *logic) Err() error {
	return l.err
}

func (l *logic) Not() Condition {
	c := &logic{
		err:  l.err,
		and:  !l.and,
		cond: make([]Condition, len(l.cond)),
	}
	for k, v := range l.cond {
		c.cond[k] = v.Not()
	}
	return c
}

func (l *logic) And(a ...Condition) Condition {
	if l.and {
		return &logic{
			err:  l.err,
			and:  true,
			cond: append(append([]Condition{}, l.cond...), a...),
		}
	}
	return And(append([]Condition{l}, a...)...)
}

func (l *logic) Or(a ...Condition) Condition {
	if !l.and {
		return &logic{
			err:  l.err,
			and:  false,
			cond: append(append([]Condition{}, l.cond...), a...),
		}
	}
	return Or(append([]Condition{l}, a...)...)
}

func (l *logic) Expand(s Starter) (string, []interface{}, error) {
	if l.err != nil {
		return "", nil, l.err
	}

	a := make([]string, len(l.cond))
	b := make([]interface{}, 0)
	for k, v := range l.cond {
		c, d, err := v.Expand(s)
		if err != nil {
			return "", nil, err
		}
		a[k] = "(" + c + ")"
		b = append(b, d...)
	}

	if l.and {
		return strings.Join(a, " AND "), b, nil
	}
	return strings.Join(a, " OR "), b, nil
}

// same as Expr
func Cond(format string, args ...interface{}) Condition {
	return &cond{
		not:  false,
		expr: newExpr(format, args...),
	}
}

// same as Cond, but logical not
func Not(format string, args ...interface{}) Condition {
	return &cond{
		not:  true,
		expr: newExpr(format, args...),
	}
}

// logical and conditions
func And(a ...Condition) Condition {
	return newLogic(true, a...)
}

// logical or conditions
func Or(a ...Condition) Condition {
	return newLogic(false, a...)
}

func newLogic(and bool, a ...Condition) Condition {
	if len(a) == 1 {
		return a[0]
	}

	c := &logic{and: and, cond: a}

	for _, i := range a {
		if err := i.Err(); err != nil {
			c.err = err
			break
		}
	}

	return c
}
