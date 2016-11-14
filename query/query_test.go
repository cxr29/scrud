// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package query

import "testing"

func TestCreate(t *testing.T) {
	for _, s := range []Starter{
		new(MySQL),
		new(Postgres),
		new(Sqlite),
	} {
		q, a, err := Insert("t1").Columns(
			"c1", "c2", "c3",
		).Values(
			false, 0, "v1",
		).Values(
			true, 1, "v2",
		).Expand(s)
		if err != nil {
			t.Fatal(err)
		}
		if q != map[string]string{
			"mysql":    "INSERT INTO `t1` (`c1`,`c2`,`c3`) VALUES (?,?,?),(?,?,?)",
			"postgres": `INSERT INTO "t1" ("c1","c2","c3") VALUES ($1,$2,$3),($4,$5,$6)`,
			"sqlite":   `INSERT INTO "t1" ("c1","c2","c3") VALUES (?,?,?),(?,?,?)`,
		}[s.DriverName()] {
			t.Fatal(s.DriverName(), "create query")
		}
		if len(a) != 6 ||
			a[0].(bool) != false ||
			a[1].(int) != 0 ||
			a[2].(string) != "v1" ||
			a[3].(bool) != true ||
			a[4].(int) != 1 ||
			a[5].(string) != "v2" {
			t.Fatal(s.DriverName(), "create argument")
		}
	}
}

func TestRetrieve(t *testing.T) {
	for _, s := range []Starter{
		new(MySQL),
		new(Postgres),
		new(Sqlite),
	} {
		q, a, err := Select(
			"t1.c1",
			"t2..c2",
			Expr("COUNT(DISTINCT `t3.c3`) AS `count`"),
		).From(
			"t1", "t2",
		).LeftJoin(
			"t3", "c4", "c5",
		).RightJoin(
			"t4", Or(Cond("`t3.c6`=`t4.c7`"), Cond("`t3.c8`>`t4.c9`")),
		).FullJoin(
			As(Select().From("t5"), "t6"),
		).Where(
			Eq("t1.c1", 1),
			In("t2..c2", "v1", "v2", "v3"),
			IsNull("t3.c6"),
		).GroupBy(
			"t4.c7",
			Expr("`t6.c6` % ?", 2),
		).Having(
			Le("count", 3).Not(),
		).OrderBy(
			Desc("t3.c8"),
			Expr("EXTRACT(YEAR FROM `t4.c9`)"),
		).Limit(4).Offset(5).Expand(s)
		if err != nil {
			t.Fatal(err)
		}
		if q != map[string]string{
			"mysql":    "SELECT `t1`.`c1`,`t2.c2`,COUNT(DISTINCT `t3`.`c3`) AS `count` FROM `t1`,`t2` LEFT JOIN `t3` USING (`c4`,`c5`) RIGHT JOIN `t4` ON (`t3`.`c6`=`t4`.`c7`) OR (`t3`.`c8`>`t4`.`c9`) FULL JOIN (SELECT * FROM `t5`) AS `t6` WHERE (`t1`.`c1`=?) AND (`t2.c2` IN (?,?,?)) AND (`t3`.`c6` IS NULL) GROUP BY `t4`.`c7`,`t6`.`c6` % ? HAVING NOT (`count`<=?) ORDER BY `t3`.`c8` DESC,EXTRACT(YEAR FROM `t4`.`c9`) LIMIT 4 OFFSET 5",
			"postgres": `SELECT "t1"."c1","t2.c2",COUNT(DISTINCT "t3"."c3") AS "count" FROM "t1","t2" LEFT JOIN "t3" USING ("c4","c5") RIGHT JOIN "t4" ON ("t3"."c6"="t4"."c7") OR ("t3"."c8">"t4"."c9") FULL JOIN (SELECT * FROM "t5") AS "t6" WHERE ("t1"."c1"=$1) AND ("t2.c2" IN ($2,$3,$4)) AND ("t3"."c6" IS NULL) GROUP BY "t4"."c7","t6"."c6" % $5 HAVING NOT ("count"<=$6) ORDER BY "t3"."c8" DESC,EXTRACT(YEAR FROM "t4"."c9") LIMIT 4 OFFSET 5`,
			"sqlite":   `SELECT "t1"."c1","t2.c2",COUNT(DISTINCT "t3"."c3") AS "count" FROM "t1","t2" LEFT JOIN "t3" USING ("c4","c5") RIGHT JOIN "t4" ON ("t3"."c6"="t4"."c7") OR ("t3"."c8">"t4"."c9") FULL JOIN (SELECT * FROM "t5") AS "t6" WHERE ("t1"."c1"=?) AND ("t2.c2" IN (?,?,?)) AND ("t3"."c6" IS NULL) GROUP BY "t4"."c7","t6"."c6" % ? HAVING NOT ("count"<=?) ORDER BY "t3"."c8" DESC,EXTRACT(YEAR FROM "t4"."c9") LIMIT 4 OFFSET 5`,
		}[s.DriverName()] {
			t.Fatal(s.DriverName(), "retrieve query", q)
		}
		if len(a) != 6 ||
			a[0].(int) != 1 ||
			a[1].(string) != "v1" ||
			a[2].(string) != "v2" ||
			a[3].(string) != "v3" ||
			a[4].(int) != 2 ||
			a[5].(int) != 3 {
			t.Fatal(s.DriverName(), "retrieve argument")
		}
	}
}

func TestUpdate(t *testing.T) {
	for _, s := range []Starter{
		new(MySQL),
		new(Postgres),
		new(Sqlite),
	} {
		q, a, err := Update("t1").Set("c1", true).Set("c2", 1).Set("c3", "v1").Where(
			Contains("c4", `\%_`),
			Between("c5", 2, 3),
		).OrderBy(Asc("c6")).Limit(4).Expand(s)
		if err != nil {
			t.Fatal(err)
		}
		if q != map[string]string{
			"mysql":    "UPDATE `t1` SET `c1`=?,`c2`=?,`c3`=? WHERE (`c4` LIKE ?) AND (`c5` BETWEEN ? AND ?) ORDER BY `c6` ASC LIMIT 4",
			"postgres": `UPDATE "t1" SET "c1"=$1,"c2"=$2,"c3"=$3 WHERE ("c4" LIKE $4) AND ("c5" BETWEEN $5 AND $6) ORDER BY "c6" ASC LIMIT 4`,
			"sqlite":   `UPDATE "t1" SET "c1"=?,"c2"=?,"c3"=? WHERE ("c4" LIKE ?) AND ("c5" BETWEEN ? AND ?) ORDER BY "c6" ASC LIMIT 4`,
		}[s.DriverName()] {
			t.Fatal(s.DriverName(), "update query")
		}
		if len(a) != 6 ||
			a[0].(bool) != true ||
			a[1].(int) != 1 ||
			a[2].(string) != "v1" ||
			a[3].(string) != `%\\\%\_%` ||
			a[4].(int) != 2 ||
			a[5].(int) != 3 {
			t.Fatal(s.DriverName(), "update argument")
		}
	}
}

func TestDelete(t *testing.T) {
	for _, s := range []Starter{
		new(MySQL),
		new(Postgres),
		new(Sqlite),
	} {
		q, a, err := Delete("t1").Where(
			And(HasPrefix("c1", "v1"), Gt("c2", 1)).Not(),
			IsNull("c3"),
		).OrderBy(Desc("c4")).Limit(2).Expand(s)
		if err != nil {
			t.Fatal(err)
		}
		if q != map[string]string{
			"mysql":    "DELETE FROM `t1` WHERE ((NOT (`c1` LIKE ?)) OR (NOT (`c2`>?))) AND (`c3` IS NULL) ORDER BY `c4` DESC LIMIT 2",
			"postgres": `DELETE FROM "t1" WHERE ((NOT ("c1" LIKE $1)) OR (NOT ("c2">$2))) AND ("c3" IS NULL) ORDER BY "c4" DESC LIMIT 2`,
			"sqlite":   `DELETE FROM "t1" WHERE ((NOT ("c1" LIKE ?)) OR (NOT ("c2">?))) AND ("c3" IS NULL) ORDER BY "c4" DESC LIMIT 2`,
		}[s.DriverName()] {
			t.Fatal(s.DriverName(), "delete query")
		}
		if len(a) != 2 ||
			a[0].(string) != "v1%" ||
			a[1].(int) != 1 {
			t.Fatal(s.DriverName(), "delete argument")
		}
	}
}
