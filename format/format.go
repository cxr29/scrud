// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Global name format functions
package format

import "unicode"

var (
	// struct name
	// return table name
	TableName = func(s string) string {
		return s
	}
	// field name, struct name, table name
	// if relation's struct name, primary key field name, table name and primary key column name
	// return column name
	ColumnName = func(a ...string) string {
		if len(a) == 3 {
			return a[0]
		} else {
			return a[3] + a[4]
		}
	}
	// field name, left struct name, left table name, right struct name, right table name
	// return many to many table name
	ManyToManyTableName = func(a ...string) string {
		if a[1] > a[3] {
			return a[3] + a[1]
		} else {
			return a[1] + a[3]
		}
	}
)

// ignore camel's underline, first upper not add underline
func CamelToUnderline(s string) string {
	a := []rune{}
	for _, r := range s {
		if r != '_' {
			if unicode.IsUpper(r) {
				a = append(a, '_', unicode.ToLower(r))
			} else {
				a = append(a, r)
			}
		}
	}
	if n := len(a); n > 0 && a[0] == '_' {
		a = a[1:]
	}
	return string(a)
}

// more than one underline as one, first always upper
func UnderlineToCamel(s string) string {
	a := []rune{}
	f := true
	for _, r := range s {
		if r == '_' {
			f = true
		} else if f {
			f = false
			a = append(a, unicode.ToUpper(r))
		} else {
			a = append(a, unicode.ToLower(r))
		}
	}
	return string(a)
}
