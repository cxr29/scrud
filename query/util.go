// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import (
	"strings"
)

func Ints2Interfaces(a []int) (b []interface{}) {
	if a != nil {
		b := make([]interface{}, len(a))
		for k, v := range a {
			b[k] = v
		}
	}
	return
}

func Strings2Interfaces(a []string) (b []interface{}) {
	if a != nil {
		b := make([]interface{}, len(a))
		for k, v := range a {
			b[k] = v
		}
	}
	return
}

func escape(s, a string) string {
	if strings.ContainsAny(s, a) {
		b := make([]rune, 0)
		for _, r := range s {
			if strings.ContainsRune(a, r) {
				b = append(b, '\\')
			}
			b = append(b, r)
		}
		return string(b)
	}
	return s
}

func EscapeRegexp(s string) string {
	return escape(s, `\.+*?()|[]{}^$`)
}

func EscapeLike(s string) string {
	return escape(s, `\_%`)
}

func quote(s, q string) string {
	if strings.Contains(s, q) {
		s = strings.Replace(s, q, q+q, -1)
	}
	return q + s + q
}

func DoubleQuote(s string) string {
	return quote(s, `"`)
}

func BackQuote(s string) string {
	return quote(s, "`")
}

func RepeatMarker(n int) string {
	s := strings.Repeat("?,", n)
	if n := len(s); n > 0 {
		s = s[:n-1]
	}
	return s
}
