// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import (
	"bytes"
	"errors"
	"strconv"
)

type join struct {
	way      string
	table    interface{}
	condtion []interface{}
}

type retrieve struct {
	alias               string
	join                []join
	elect, group, order []interface{}
	from                []interface{}
	where, having       []Condition
	limit, offset       int
}

// select clause generator
//
// string or expression
func Select(a ...interface{}) *retrieve {
	r := new(retrieve)
	r.elect = append(r.elect, a...)
	return r
}

func (r *retrieve) Err() error {
	return nil
}

func (r *retrieve) Expand(s Starter) (string, []interface{}, error) {
	buf, args := new(bytes.Buffer), make([]interface{}, 0)

	buf.WriteString("SELECT ")

	if n := len(r.elect); n > 0 {
		for k, v := range r.elect {
			var x Expression
			switch i := v.(type) {
			case string:
				x = Expr(quote(i))
			case Expression:
				x = i
			default:
				return "", nil, errors.New("retrieve: select must be string or expression")
			}
			e, a, err := x.Expand(s)
			if err != nil {
				return "", nil, err
			}
			buf.WriteString(e)
			args = append(args, a...)
			if k < n-1 {
				buf.WriteByte(',')
			}
		}
	} else {
		buf.WriteString("*")
	}

	if n := len(r.from); n > 0 {
		buf.WriteString(" FROM ")
		for k, v := range r.from {
			switch i := v.(type) {
			case string:
				buf.WriteString(s.FormatName(i))
			case Querier:
				e, a, err := i.Expand(s)
				if err != nil {
					return "", nil, err
				}
				buf.WriteByte('(')
				buf.WriteString(e)
				buf.WriteString(") AS ")
				buf.WriteString(s.FormatName(i.Alias()))
				args = append(args, a...)
			default:
				return "", nil, errors.New("retrieve: from must be string or querier")
			}
			if k < n-1 {
				buf.WriteByte(',')
			}
		}
	} else {
		return "", nil, errors.New("retrieve: empty from")
	}

	if n := len(r.join); n > 0 {
		for _, j := range r.join {
			buf.WriteByte(' ')
			buf.WriteString(j.way)
			buf.WriteByte(' ')

			switch i := j.table.(type) {
			case string:
				buf.WriteString(s.FormatName(i))
			case Querier:
				e, a, err := i.Expand(s)
				if err != nil {
					return "", nil, err
				}
				buf.WriteByte('(')
				buf.WriteString(e)
				buf.WriteString(") AS ")
				buf.WriteString(s.FormatName(i.Alias()))
				args = append(args, a...)
			default:
				return "", nil, errors.New("retrieve: join must be string or querier")
			}

			using, on := make([]string, 0), make([]Condition, 0)
			for _, c := range j.condtion {
				switch i := c.(type) {
				case string:
					using = append(using, i)
				case Condition:
					on = append(on, i)
				default:
					return "", nil, errors.New("retrieve: join using string or on condition")
				}
			}

			if x, y := len(using), len(on); x > 0 && y > 0 {
				return "", nil, errors.New("retrieve: join using string or on condition, but not both")
			} else if x > 0 {
				buf.WriteString(" USING (")
				for k, v := range using {
					buf.WriteString(s.FormatName(v))
					if k < x-1 {
						buf.WriteByte(',')
					}
				}
				buf.WriteByte(')')
			} else if y > 0 {
				buf.WriteString(" ON ")
				e, a, err := And(on...).Expand(s)
				if err != nil {
					return "", nil, err
				}
				buf.WriteString(e)
				args = append(args, a...)
			}
		}
	}

	if n := len(r.where); n > 0 {
		buf.WriteString(" WHERE ")
		e, a, err := And(r.where...).Expand(s)
		if err != nil {
			return "", nil, err
		}
		buf.WriteString(e)
		args = append(args, a...)
	}

	if n := len(r.group); n > 0 {
		buf.WriteString(" GROUP BY ")
		for k, v := range r.group {
			var x Expression
			switch i := v.(type) {
			case string:
				x = Expr(quote(i))
			case Expression:
				x = i
			default:
				return "", nil, errors.New("retrieve: group by must be string or expression")
			}
			e, a, err := x.Expand(s)
			if err != nil {
				return "", nil, err
			}
			buf.WriteString(e)
			args = append(args, a...)
			if k < n-1 {
				buf.WriteByte(',')
			}
		}
	}

	if n := len(r.having); n > 0 {
		buf.WriteString(" HAVING ")
		e, a, err := And(r.having...).Expand(s)
		if err != nil {
			return "", nil, err
		}
		buf.WriteString(e)
		args = append(args, a...)
	}

	if n := len(r.order); n > 0 {
		buf.WriteString(" ORDER BY ")
		for k, v := range r.order {
			var x Expression
			switch i := v.(type) {
			case string:
				x = Expr(quote(i))
			case Expression:
				x = i
			default:
				return "", nil, errors.New("retrieve: order by must be string or expression")
			}
			e, a, err := x.Expand(s)
			if err != nil {
				return "", nil, err
			}
			buf.WriteString(e)
			args = append(args, a...)
			if k < n-1 {
				buf.WriteByte(',')
			}
		}
	}

	if r.limit > 0 {
		buf.WriteString(" LIMIT ")
		buf.WriteString(strconv.Itoa(r.limit))
	}

	if r.offset > 0 {
		buf.WriteString(" OFFSET ")
		buf.WriteString(strconv.Itoa(r.offset))
	}

	return buf.String(), args, nil
}

func (r *retrieve) Alias() string {
	return r.alias
}

func (r *retrieve) As(s string) Querier {
	r.alias = s
	return r
}

// string or expression
func (r *retrieve) Select(a ...interface{}) *retrieve {
	r.elect = append(r.elect, a...)
	return r
}

// string or querier
func (r *retrieve) From(a ...interface{}) *retrieve {
	r.from = append(r.from, a...)
	return r
}

// t: string or querier, a: either using string or on condition
func (r *retrieve) Join(w string, t interface{}, a ...interface{}) *retrieve {
	r.join = append(r.join, join{w, t, a})
	return r
}

func (r *retrieve) InnerJoin(t interface{}, a ...interface{}) *retrieve {
	return r.Join("INNER JOIN", t, a...)
}

func (r *retrieve) LeftJoin(t interface{}, a ...interface{}) *retrieve {
	return r.Join("LEFT JOIN", t, a...)
}

func (r *retrieve) RightJoin(t interface{}, a ...interface{}) *retrieve {
	return r.Join("RIGHT JOIN", t, a...)
}

func (r *retrieve) FullJoin(t interface{}, a ...interface{}) *retrieve {
	return r.Join("FULL JOIN", t, a...)
}

func (r *retrieve) Where(a ...Condition) *retrieve {
	r.where = append(r.where, a...)
	return r
}

// string or expression
func (r *retrieve) GroupBy(a ...interface{}) *retrieve {
	r.group = append(r.group, a...)
	return r
}

func (r *retrieve) Having(a ...Condition) *retrieve {
	r.having = append(r.having, a...)
	return r
}

// string or expression
func (r *retrieve) OrderBy(a ...interface{}) *retrieve {
	r.order = append(r.order, a...)
	return r
}

func (r *retrieve) Limit(n int) *retrieve {
	r.limit = n
	return r
}

func (r *retrieve) Offset(n int) *retrieve {
	r.offset = n
	return r
}
