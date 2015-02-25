scrud - Go struct/SQL CRUD & Go write SQL
===

想要的几乎都有，如果没有的话，请[提交请求](https://github.com/cxr29/scrud/pulls)

### 安装
```go get github.com/cxr29/scrud```

### CRUD
```Go
import "github.com/cxr29/scrud"
import _ "github.com/go-sql-driver/mysql"

db, err := scrud.Open("mysql", "user:password@/database")

// A, B is struct or *struct
n, err := db.Insert(A)                // insert
n, err = db.Insert([]A{})             // batch insert
err = db.Select(&A, ...)              // select by primary key, support include or exclude columns
err = db.SelectRelation("B", &A, ...) // select relation field, support include or exclude columns
err = db.Update(A, ...)               // update by primary key, support include or exclude columns
err = db.Delete(A)                    // delete by primary key

m2m := db.ManyToMany("B", A) // many to many field manager
err = m2m.Add(B, ...)        // add relation
err = m2m.Set(B, ...)        // set relation, empty other
err = m2m.Remove(B, ...)     // remove relation
has, err := m2m.Has(B)       // check relation
err = m2m.Empty()            // empty relation

result, err := db.Run(qe)      // run a query expression that doesn't return rows
err = db.Fetch(qe).One(&A)     // run a query expression and fetch one row to struct
err = db.Fetch(qe).All(&[]A{}) // run a query expression and fetch rows to slice of struct
```

### SQL
```Go
import . "github.com/cxr29/scrud/query"

// Create
query, args, err := Insert("Table1").Columns(
	"Column1", "Column2", "Column3", ...
).Values(
	false, 0, "hi", ...
).Values(
	true, 1, "cxr", ...
).Expand(new(MySQL))

// Retrieve
query, args, err = Select(
	"Column1",
	"Table1.Column2",                             // `Table1`.`Column1`
	Expr("COUNT(DISTINCT `Column3`) AS `Count`"), // -- expression also ok
	...
).From("Table1", ...).LeftJoin("Table2", // LEFT JOIN `Table2` USING (`Column2`, ...)
	"Column2", // -- join using when string
	...
).RightJoin("Table3", // LEFT JOIN `Table3` ON `Column4`=? AND `Table1`.`Column1`+`Table3`.`Column1`>? ... -- true, 1
	Eq("Column4", true),                            // -- join on when Condition
	Cond("`Table1.Column1`+`Table3.Column1`>?", 1), // -- AND
	...
).FullJoin(Select().From(`Table4`).As("a"), // (SELECT * FROM `Table4`) AS `a` -- join subquery
	...
).Where( // -- AND
	In("Column5", 2, 3, 4),    // `Column5` IN (?,?,?) -- 2,3,4
	Contains("Column6", "_"), // `Column6` LIKE ? -- %\_%
	Cond("`Table3.Column3` > `a.Column4`"),
	...
).GroupBy(
	"Column1",
	Expr("EXTRACT(YEAR FROM Column7`)"), // -- expression also ok
).Having( // -- same as Where
	...
).OrderBy(
	Desc("Column1"),                     // `Column1` DESC
	Expr("EXTRACT(YEAR FROM Column7`)"), // -- expression also ok
).Limit(5).Offset(6).Expand(new(Postgres))

// Update
query, args, err = Update("Table1").Set(
	"Column1", true,
).Set( // -- expression also ok
	"Column2", Expr("Column2+?", 1), // `Column2`=`Column2`+? -- 1
).Where(
	... // -- same as Retrieve Where
).OrderBy(
	... // -- same as Retrieve OrderBy
).Limit(...).Expand(new(Sqlite))

// Delete
query, args, err = Delete("Table1").Where(
	... // -- same as Retrieve Where
).OrderBy(
	... // -- same as Retrieve OrderBy
).Limit(...).Expand(...)
```

### [Expression](http://godoc.org/github.com/cxr29/scrud/query#Expression)和[Condition](http://godoc.org/github.com/cxr29/scrud/query#Condition)
像写原生SQL一样轻松编写任何SQL语句，自动处理标识符及参数

### 文档
API文档位于：[http://godoc.org/github.com/cxr29/scrud](http://godoc.org/github.com/cxr29/scrud)

### 结构体定义
`      标签：`\`scrud:"ColumnName,option,..."\`  
`多对多标签：`\`scrud:"TableName|LeftColumnName|RigthColumnName,option,..."\`  
`  标签选项：`primary_key, auto_increment, auto_now_add, auto_now, one_to_one, one_to_many, many_to_one/foreign_key, many_to_many  

### 命名
`      表：Tabler    > `[format.TableName](http://godoc.org/github.com/cxr29/scrud/format#pkg-variables)  
`      列：field tag > Columner > `[format.ColumnName](http://godoc.org/github.com/cxr29/scrud/format#pkg-variables)  
`  关系列：field tag > Columner > `[format.ColumnName](http://godoc.org/github.com/cxr29/scrud/format#pkg-variables)  
`多对多表：field tag > `[format.ManyToManyTableName](http://godoc.org/github.com/cxr29/scrud/format#pkg-variables)  

```Go
type Tabler interface {
	// return table name
	TableName() string
}

type Columner interface {
	// field name
	// if relation's struct name, primary key field name, table name and primary key column name
	// return column name
	ColumnName(...string) string
}
```

由于无法区分方法是否为嵌入方法，所以不能嵌入的关系结构体实现了相关接口，但被嵌入结构体未实现  

### 自动
```
自动递增(auto_increment) ints或uints
创建时间(auto_now_add)   time.Time   当插入时
修改时间(auto_now)       time.Time   当插入和更新时
```
### 主键
仅支持单主键(primary_key)  
如果没有主键，则自动递增默认为主键  
如果没有主键和自动递增，则名为`Id`且类型为ints或uints的字段设置为主键和自动递增  

### 关系
```
一对一(one_to_one)                   反向关系 一对一
一对多(one_to_many)                  反向关系 多对一/外键
多对一/外键(many_to_one/foreign_key) 反向关系 一对多
多对多(many_to_many)                 反向关系 多对多
```
```Go
// 一对一，则B表必须有主键，A表含B表主键；反之亦然
type A struct {
	B `,one_to_one`
}

// 一对多，则A表必须有主键，B表含A表主键；反之多对一
type A struct {
	X []B `,one_to_many`
}

// 多对一，则B表必须有主键，A表含B表主键；反之一对多
type A struct {
	B `,many_to_one`
}

// 多对多，则A表和B表都必须有主键，默认使用A表主键和B表主键生成一个中间表；反之亦然
type A struct {
	X []B `,many_to_many`
}

type Througher interface {
	// field name
	// return through table, left field name, right field name
	ThroughTable(string) (interface{}, string, string)
}

// 多对多通过表，C表必须同时将A表和B表设为多对一关系
type C struct {
	A `,foreign_key`
	B `,foreign_key`
	...
}

func (_ *A) ThroughTable(field string) (interface{}, string, string) {
	return new(C), "A", "B"
}
```

### Getter和Setter
```
ScrudGet**() $$ 或 ScrudGet**() ($$, error)
ScrudSet**($$)  或 ScrudSet**($$) error
```
`**`表示字段名  
`$$`表示参数或返回值，类型必须是：`int64/float64/bool/[]byte/string/time.Time/interface{}`  
Setter必须是指针接收器，自动和关系字段不允许Getter和Setter  

### 欢迎更多测试，[功能特性](https://github.com/cxr29/scrud/pulls)，[问题反馈](https://github.com/cxr29/scrud/issues)...
