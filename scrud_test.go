// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

// set data source name environment variable TestMySQLScrud to test mysql
package scrud

import (
	"os"
	"testing"
	"time"

	"github.com/cxr29/scrud/format"
	. "github.com/cxr29/scrud/query"
	_ "github.com/go-sql-driver/mysql"
)

type Row struct {
	Id uint
	C1 bool
	C2 int
	CS string
	CT time.Time `,auto_now`
}

func (_ *Row) TableName() string {
	return "scrud_row"
}

func (_ *Row) ColumnName(a ...string) string {
	return format.CamelToUnderline(a[0])
}

func TestMySQLScrud(t *testing.T) {
	dsn := os.Getenv("TestMySQLScrud")
	if dsn == "" {
		t.SkipNow()
	}

	db, err := Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_, err = db.Exec("DROP TABLE `scrud_row`")
		if err != nil {
			t.Fatal(err)
		}
	}()

	_, err = db.Exec("CREATE TABLE `scrud_row` (`id` INT UNSIGNED NOT NULL AUTO_INCREMENT, `c1` TINYINT NOT NULL, `c2` INT NOT NULL, `c_s` VARCHAR(255) NOT NULL, `c_t` DATETIME NOT NULL, PRIMARY KEY (`id`))")
	if err != nil {
		t.Fatal(err)
	}

	r1 := &Row{CS: "cxr"}
	if _, err := db.Insert(r1); err != nil {
		t.Fatal(err)
	}
	if r1.Id == 0 || r1.CT.IsZero() {
		t.Fatal("insert")
	}

	r1.C1, r1.C2 = true, 2
	if err := db.Update(r1, "C1", "c2"); err != nil {
		t.Fatal(err)
	}

	r2 := &Row{Id: r1.Id}
	if err := db.Select(r2); err != nil {
		t.Fatal(err)
	}

	if r2.C1 != r1.C1 || r2.C2 != r1.C2 || r2.CS != r1.CS || r2.CT.Unix() != r1.CT.Unix() {
		t.Fatal("select")
	}

	if err := db.Delete(r2); err != nil {
		t.Fatal(err)
	}

	r1, r2 = &Row{C1: false, C2: 1, CS: "hello"}, &Row{C1: true, C2: 2, CS: "world"}

	a := []*Row{r2, r1}
	if _, err := db.Insert(a); err != nil {
		t.Fatal(err)
	}
	if r1.Id != 0 || r2.Id != 0 || r1.CT.IsZero() || r2.CT.IsZero() {
		t.Fatal("batch insert")
	}

	if err := db.Fetch(
		Select().From("scrud_row").OrderBy("c_s"),
	).All(&a); err != nil {
		t.Fatal(err)
	}

	if len(a) != 2 ||
		a[0].Id == 0 || a[0].C1 != r1.C1 || a[0].C2 != r1.C2 || a[0].CS != r1.CS || a[0].CT.Unix() != r1.CT.Unix() ||
		a[1].Id == 0 || a[1].C1 != r2.C1 || a[1].C2 != r2.C2 || a[1].CS != r2.CS || a[1].CT.Unix() != r2.CT.Unix() {
		t.Fatal("fetch all")
	}
}

type Node struct {
	Id              int
	Parent          *Node   `ParentId,foreign_key`
	Children        []*Node `scrud:"ParentId,one_to_many"`
	Siblings        []*Node `scrud:"ScrudNodeSibling|LeftId|RightId,many_to_many"`
	ThroughSiblings []*Node `scrud:",many_to_many"`
	Data            string
	Time            time.Time `,auto_now_add`
}

func (_ *Node) TableName() string {
	return "ScrudNode"
}

type ScrudNodeSibling struct {
	Left  *Node `LeftId,many_to_one`
	Right *Node `RightId,foreign_key`
}

func (_ *Node) ThroughTable(field string) (interface{}, string, string) {
	if field == "ThroughSiblings" {
		return new(ScrudNodeSibling), "Left", "Right"
	}
	return nil, "", ""
}

func TestMySQLScrudNode(t *testing.T) {
	dsn := os.Getenv("TestMySQLScrud")
	if dsn == "" {
		t.SkipNow()
	}

	db, err := Open("mysql", dsn)
	if err != nil {
		t.Fatal(err)
	}

	defer func() {
		_, err = db.Exec("DROP TABLE `ScrudNode`")
		if err != nil {
			t.Fatal(err)
		}
		_, err = db.Exec("DROP TABLE `ScrudNodeSibling`")
		if err != nil {
			t.Fatal(err)
		}
	}()

	_, err = db.Exec("CREATE TABLE `ScrudNode` (`Id` INT NOT NULL AUTO_INCREMENT, `ParentId` INT NOT NULL, `Data` VARCHAR(255) NOT NULL, `Time` DATETIME NOT NULL, PRIMARY KEY (`Id`))")
	if err != nil {
		t.Fatal(err)
	}
	_, err = db.Exec("CREATE TABLE `ScrudNodeSibling` (`LeftId` INT NOT NULL, `RightId` INT NOT NULL)")
	if err != nil {
		t.Fatal(err)
	}

	node1 := &Node{Parent: new(Node), Data: "cxr1"}
	_, err = db.Insert(node1)
	if err != nil {
		t.Fatal(err)
	}

	node2 := &Node{Parent: node1, Data: "cxr2"}
	node3 := &Node{Parent: node1, Data: "cxr3"}
	node4 := &Node{Parent: node1, Data: "cxr4"}
	_, err = db.Insert([]*Node{node2, node3, node4}) // no Id
	if err != nil {
		t.Fatal(err)
	}

	node2.Parent = &Node{Id: node1.Id}
	err = db.SelectRelation("Parent", node2)
	if err != nil {
		t.Fatal(err)
	}
	if node2.Parent.Data != "cxr1" || node2.Parent.Time.IsZero() {
		t.Fatal("select relation many_to_one/foreign_key")
	}

	err = db.SelectRelation("Children", node1) // cxr? OrderBy
	if err != nil {
		t.Fatal(err)
	}

	if len(node1.Children) != 3 ||
		node1.Children[0].Data != "cxr2" ||
		node1.Children[1].Data != "cxr3" ||
		node1.Children[2].Data != "cxr4" {
		t.Fatal("select relation one_to_many")
	}

	// for Id
	node2 = node1.Children[0]
	node3 = node1.Children[1]
	node4 = node1.Children[2]

	m2m := db.ManyToMany("Siblings", node3)

	if err := m2m.Set(node2); err != nil {
		t.Fatal(err)
	}
	if has, err := m2m.Has(node2); err != nil {
		t.Fatal(err)
	} else if !has {
		t.Fatal("many_to_many has")
	}

	m2m = db.ManyToMany("ThroughSiblings", node3)

	if has, err := m2m.Has(node4); err != nil {
		t.Fatal(err)
	} else if has {
		t.Fatal("many_to_many through has")
	}
	if err = m2m.Add(node4); err != nil {
		t.Fatal(err)
	}

	err = db.SelectRelation("Siblings", node3) // cxr? OrderBy
	if err != nil {
		t.Fatal(err)
	}

	if len(node3.Siblings) != 2 ||
		node3.Siblings[0].Data != "cxr2" ||
		node3.Siblings[1].Data != "cxr4" {
		t.Fatal("select relation many_to_many")
	}

	err = db.SelectRelation("ThroughSiblings", node3) // cxr? OrderBy
	if err != nil {
		t.Fatal(err)
	}

	if len(node3.ThroughSiblings) != 2 ||
		node3.ThroughSiblings[0].Data != "cxr2" ||
		node3.ThroughSiblings[1].Data != "cxr4" {
		t.Fatal("select relation many_to_many through")
	}

	err = m2m.Remove(node2, node4)
	if err != nil {
		t.Fatal(err)
	}
	if has, err := m2m.Has(node2); err != nil {
		t.Fatal(err)
	} else if has {
		t.Fatal("many_to_many remove")
	}
}
