// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// Go write SQL
//
//  import . "github.com/cxr29/scrud/query"
//
//  // Create
//  query, args, err := Insert("Table1").Columns(
//  	"Column1", "Column2", "Column3", ...
//  ).Values(
//  	false, 0, "hi", ...
//  ).Values(
//  	true, 1, "cxr", ...
//  ).Expand(new(MySQL))
//
//  // Retrieve
//  query, args, err = Select(
//  	"Column1",
//  	"Table1.Column2",                             // `Table1`.`Column1`
//  	Expr("COUNT(DISTINCT `Column3`) AS `Count`"), // -- expression also ok
//  	...
//  ).From("Table1", ...).LeftJoin("Table2", // LEFT JOIN `Table2` USING (`Column2`, ...)
//  	"Column2", // -- join using when string
//  	...
//  ).RightJoin("Table3", // LEFT JOIN `Table3` ON `Column4`=? AND `Table1`.`Column1`+`Table3`.`Column1`>? ... -- true, 1
//  	Eq("Column4", true),                            // -- join on when Condition
//  	Cond("`Table1.Column1`+`Table3.Column1`>?", 1), // -- AND
//  	...
//  ).FullJoin(Select().From(`Table4`).As("a"), // (SELECT * FROM `Table4`) AS `a` -- join subquery
//  	...
//  ).Where( // -- AND
//  	In("Column5", 2, 3, 4),    // `Column5` IN (?,?,?) -- 2,3,4
//  	Contains("Column6", "_"), // `Column6` LIKE ? -- %\_%
//  	Cond("`Table3.Column3` > `a.Column4`"),
//  	...
//  ).GroupBy(
//  	"Column1",
//  	Expr("EXTRACT(YEAR FROM Column7`)"), // -- expression also ok
//  ).Having( // -- same as Where
//  	...
//  ).OrderBy(
//  	Desc("Column1"),                     // `Column1` DESC
//  	Expr("EXTRACT(YEAR FROM Column7`)"), // -- expression also ok
//  ).Limit(5).Offset(6).Expand(new(Postgres))
//
//  // Update
//  query, args, err = Update("Table1").Set(
//  	"Column1", true,
//  ).Set( // -- expression also ok
//  	"Column2", Expr("Column2+?", 1), // `Column2`=`Column2`+? -- 1
//  ).Where(
//  	... // -- same as Retrieve Where
//  ).OrderBy(
//  	... // -- same as Retrieve OrderBy
//  ).Limit(...).Expand(new(Sqlite))
//
//  // Delete
//  query, args, err = Delete("Table1").Where(
//  	... // -- same as Retrieve Where
//  ).OrderBy(
//  	... // -- same as Retrieve OrderBy
//  ).Limit(...).Expand(...)
//
// See https://github.com/cxr29/scrud for more details
package query

import (
	"strconv"
	"strings"
)

// starter expand expression, format the identifier and replace the placeholder
type Starter interface {
	DriverName() string
	NextMarker() string
	FormatName(string) string
}

// sql segments contains identifier and placeholder with arguments
type Expression interface {
	Err() error
	Expand(Starter) (string, []interface{}, error)
}

// expression with logical operation for join on, where and having
type Condition interface {
	Expression
	And(...Condition) Condition
	Or(...Condition) Condition
	Not() Condition
}

type Querier interface {
	Expression
	Alias() string
	As(string) Querier // as subquery
}

// mysql starter to expand expression
type MySQL int

func (_ *MySQL) DriverName() string {
	return "mysql"
}

func (_ *MySQL) FormatName(s string) string {
	return "`" + strings.Replace(s, "`", "``", -1) + "`"
}

func (_ *MySQL) NextMarker() string {
	return "?"
}

// postgres starter to expand expression
type Postgres int

func (_ *Postgres) DriverName() string {
	return "postgres"
}

func (x *Postgres) NextMarker() string {
	*x++
	return "$" + strconv.Itoa(int(*x))
}

func (_ *Postgres) FormatName(s string) string {
	return `"` + strings.Replace(s, `"`, `""`, -1) + `"`
}

// sqlite starter to expand expression
type Sqlite int

func (_ *Sqlite) DriverName() string {
	return "sqlite"
}

func (_ *Sqlite) FormatName(s string) string {
	return `"` + strings.Replace(s, `"`, `""`, -1) + `"`
}

func (_ *Sqlite) NextMarker() string {
	return "?"
}

// `k`=?
func Eq(k string, v interface{}) Condition {
	if v == nil {
		return IsNull(k)
	} else {
		return Cond(quote(k)+"=?", v)
	}
}

// `k`<?
func Lt(k string, v interface{}) Condition {
	return Cond(quote(k)+"<?", v)
}

// `k`<=?
func Le(k string, v interface{}) Condition {
	return Cond(quote(k)+"<=?", v)
}

// `k`>?
func Gt(k string, v interface{}) Condition {
	return Cond(quote(k)+">?", v)
}

// `k`>=?
func Ge(k string, v interface{}) Condition {
	return Cond(quote(k)+">=?", v)
}

// `k` IN (?,...)
func In(k string, a ...interface{}) Condition {
	return Cond(quote(k)+" IN ("+repeatMarker(len(a))+")", a...)
}

// `k` BETWEEN ? AND ?
func Between(k string, start, end interface{}) Condition {
	return Cond(quote(k)+" BETWEEN ? AND ?", start, end)
}

// `k` LIKE ?
func Like(k, v string) Condition {
	return Cond(quote(k)+" LIKE ?", v)
}

// `k` LIKE %?%
func Contains(k, v string) Condition {
	return Like(k, "%"+escapeLike(v)+"%")
}

// `k` LIKE ?%
func HasPrefix(k, v string) Condition {
	return Like(k, escapeLike(v)+"%")
}

// `k` LIKE %?
func HasSuffix(k, v string) Condition {
	return Like(k, "%"+escapeLike(v))
}

// `k` IS NULL
func IsNull(k string) Condition {
	return Cond(quote(k) + " IS NULL")
}

// `k` ASC
func Asc(k string) Expression {
	return Expr(quote(k) + " ASC")
}

// `k` DESC
func Desc(k string) Expression {
	return Expr(quote(k) + " DESC")
}
