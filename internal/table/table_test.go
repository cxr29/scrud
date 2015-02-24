// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package table

import (
	"reflect"
	"testing"
	"time"

	"github.com/cxr29/scrud/format"
)

type T1 struct {
	Id int
	C1 bool
	C2 int
	C3 string
}

func (t *T1) ScrudGetC3() []byte {
	return []byte(t.C3)
}

func (t *T1) ScrudSetC3(a []byte) error {
	t.C3 = string(a) + " world"
	return nil
}

func TestT1(t *testing.T) {
	t1, err := NewTable(T1{})
	if err != nil {
		t.Fatal(err)
	}
	if t1.PrimaryKey == nil || t1.AutoIncrement == nil ||
		t1.PrimaryKey.Name != "Id" || len(t1.Columns) != 4 ||
		t1.FieldMap["C3"].Getter != 0 || t1.FieldMap["C3"].GetPointer != true ||
		t1.FieldMap["C3"].Setter != 1 || t1.FieldMap["C3"].SetError != true ||
		t1.FieldMap["C3"].GetType != t1.FieldMap["C3"].SetType {
		t.Fatal("t1")
	}

	x := new(T1)
	v := reflect.ValueOf(x).Elem()
	c := t1.FieldMap["C2"]
	if err = c.SetValue(v, 1); err != nil {
		t.Fatal(err)
	}
	i, err := c.GetValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if x.C2 != 1 || i.(int) != 1 {
		t.Fatal("t1 set and get value")
	}

	c = t1.FieldMap["C3"]
	if err = c.SetValue(v, "hello"); err != nil {
		t.Fatal(err)
	}
	i, err = c.GetValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if x.C3 != "hello world" || string(i.([]byte)) != "hello world" {
		t.Fatal("t1 setter and getter")
	}
}

type T2 struct {
	*T1 `,primary_key,one_to_one`
}

func (t *T2) TableName() string {
	return "t2"
}

func TestT2(t *testing.T) {
	t2, err := NewTable(T2{})
	if err != nil {
		t.Fatal(err)
	}
	if t2.Name != "t2" || t2.PrimaryKey == nil || t2.PrimaryKey.Name != "T1Id" {
		t.Fatal("t2")
	}

	x := new(T2)
	v := reflect.ValueOf(x).Elem()
	c := t2.FieldMap["T1"]
	if err = c.SetValue(v, 1); err != nil {
		t.Fatal(err)
	}
	i, err := c.GetValue(v)
	if err != nil {
		t.Fatal(err)
	}
	if x.T1 == nil || x.T1.Id != 1 || i.(int) != 1 {
		t.Fatal("t2 set and get value")
	}
}

type T3 struct {
	*T2 `scrud:",foreign_key"`
	CC  string
}

func (t *T3) ColumnName(a ...string) string {
	if len(a) == 5 {
		return format.CamelToUnderline(a[3]) + "_" + format.CamelToUnderline(a[4])
	} else {
		return format.CamelToUnderline(a[0])
	}
}

func TestT3(t *testing.T) {
	if t3, err := NewTable(T3{}); err != nil {
		t.Fatal(err)
	} else if t3.PrimaryKey != nil || len(t3.Columns) != 2 ||
		t3.Columns[0].Name != "t2_t1_id" || t3.Columns[1].Name != "c_c" {
		t.Fatal("t3")
	}
}

type T4 struct {
	Id string `scrud:",primary_key"`
	T1 []*T1  `,many_to_many`
	C1 int    `-`
}

func TestT4(t *testing.T) {
	if t4, err := NewTable(T4{}); err != nil {
		t.Fatal(err)
	} else if len(t4.Columns) != 2 ||
		t4.Columns[0].Name != "Id" || !t4.Columns[0].PrimaryKey() ||
		t4.Columns[1].Name != "T1T4" || t4.Columns[1].NameLeft != "T4Id" || t4.Columns[1].NameRight != "T1Id" ||
		len(t4.ColumnMap) != 1 {
		t.Fatal("t4")
	}
}

type T5 struct {
	Id string    `scrud:",primary_key"`
	T2 []*T2     `,many_to_many`
	Ct time.Time `,auto_now_add`
}

type T6 struct {
	Id int
	L  *T5 `,many_to_one`
	R  *T2 `,many_to_one`
}

func (t *T5) ThroughTable(f string) (interface{}, string, string) {
	return T6{}, "L", "R"
}

func TestT5(t *testing.T) {
	if t5, err := NewTable(T5{}); err != nil {
		t.Fatal(err)
	} else if len(t5.Columns) != 3 ||
		t5.Columns[0].Name != "Id" || !t5.Columns[0].PrimaryKey() ||
		t5.Columns[1].RelationTable == nil || t5.Columns[1].RelationTable.Name != "t2" ||
		t5.Columns[1].Name != "T2T5" || t5.Columns[1].NameLeft != "T5Id" || t5.Columns[1].NameRight != "T2T1" ||
		t5.Columns[1].ThroughTable == nil || t5.Columns[1].ThroughTable.Name != "T6" ||
		t5.Columns[1].ThroughLeft.Index != 1 || t5.Columns[1].ThroughRight.Index != 2 ||
		t5.Columns[2].Name != "Ct" || !t5.Columns[2].AutoNowAdd() {
		t.Fatal("t5")
	}
}

type Node struct {
	Id       int
	Parent   *Node   `,foreign_key`
	Children []*Node `scrud:",one_to_many"`
	Siblings []*Node `TableName|LeftId|RightId,many_to_many`
}

func TestNode(t *testing.T) {
	if node, err := NewTable(Node{}); err != nil {
		t.Fatal(err)
	} else if node.PrimaryKey == nil || node.AutoIncrement == nil ||
		node.PrimaryKey.Name != "Id" ||
		len(node.Columns) != 4 || len(node.ColumnMap) != 2 ||
		node.Columns[1].Name != "NodeId" ||
		node.Columns[2].Name != "NodeId" ||
		node.Columns[3].Name != "TableName" || node.Columns[3].NameLeft != "LeftId" || node.Columns[3].NameRight != "RightId" {
		t.Fatal("node")
	}
}
