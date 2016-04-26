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

func escapeRegexp(s string) string {
	return escape(s, `\.+*?()|[]{}^$`)
}

func EscapeLike(s string) string {
	return escape(s, `\_%`)
}

func quote(s string) string {
	/*
		// cxr? whether or not
		a, dot := []rune(s), 0
		for i, n := 0, len(a); i < n; i++ {
			if r := a[i]; r == '`' {
				if j := i + 1; j < n && a[j] == r {
					i = j
				} else {
					panic("back quote must be double")
				}
			} else if r == '.' {
				if j := i + 1; j < n && a[j] == r {
					i = j
				} else {
					dot++
				}
			} else if r == '\x00' {
				panic("not allow null")
			}
		}
		if dot > 1 {
			panic("too many dot")
		}
	*/
	return "`" + s + "`"
}

func repeatMarker(n int) string {
	s := strings.Repeat("?,", n)
	if n := len(s); n > 0 {
		s = s[:n-1]
	}
	return s
}
