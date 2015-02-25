scrud - Go struct/SQL CRUD & Go write SQL
===

Almost what you want, if missing, please [pull requests](https://github.com/cxr29/scrud/pulls)

### Install
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

### [Expression](http://godoc.org/github.com/cxr29/scrud/query#Expression) and [Condition](http://godoc.org/github.com/cxr29/scrud/query#Condition)
Build any sql clause contains identifier and placeholder with arguments like raw sql

### Documentation
API documentation can be found here: [http://godoc.org/github.com/cxr29/scrud](http://godoc.org/github.com/cxr29/scrud)

### Struct Definition
```
             tag: `scrud:"ColumnName,option,..."`
many to many tag: `scrud:"TableName|LeftColumnName|RigthColumnName,option,..."`
         options: primary_key, auto_increment, auto_now_add, auto_now, one_to_one, one_to_many, many_to_one/foreign_key, many_to_many
```

### Name
```
             table: Tabler    > format.TableName
            column: field tag > Columner > format.ColumnName
   relation column: field tag > Columner > format.ColumnName
many to many table: field tag > format.ManyToManyTableName
```
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

Because can't distinguish between that a type has a method vs the type getting the method from an embedded type, so can't only define interfaces in embedded relation type  

### Auto
```
auto_increment ints or uints
auto_now_add   time.Time     when insert
auto_now       time.Time     when insert and update
```

###Primary Key
Only support single primary_key  
If no primary key, the auto_increment as primary_key  
If no primary_key and auto_increment, then the ints or uints `Id` field as primary_key and auto_increment  

###Relation
```
one_to_one              reverse one_to_one
one_to_many             reverse many_to_one/foreign_key
many_to_one/foreign_key reverse one_to_many
many_to_many            reverse many_to_many
```
```Go
// one_to_one, B must have primary key, A contians B's primary key; vice versa
type A struct {
	B `,one_to_one`
}

// one_to_many, A must have primary key, B contains A's primary key; reverse many_to_one/foreign_key
type A struct {
	X []B `,one_to_many`
}

// many_to_one/foreign_key, B must have primary key, A contians B's primary key; reverse one_to_many
type A struct {
	B `,many_to_one`
}

// many_to_many, both A and B must have primary key, A's primary key and B's primary key produce a table; vice versa
type A struct {
	X []B `,many_to_many`
}

type Througher interface {
	// field name
	// return through table, left field name, right field name
	ThroughTable(string) (interface{}, string, string)
}

// many_to_many through table, C must contains A and B as many_to_one/foreign_key
type C struct {
	A `,many_to_one`
	B `,foreign_key`
	...
}

func (_ *A) ThroughTable(field string) (interface{}, string, string) {
	return new(C), "A", "B"
}
```

### Getter and Setter
```
ScrudGet**() $$ or ScrudGet**() ($$, error)
ScrudSet**($$)  or ScrudSet**($$) error
```
`**` is field name  
`$$` is argement or return value, type should be: `int64/float64/bool/[]byte/string/time.Time/interface{}`  
Setter must be pointer receiver, auto fields and relation fields not alllow Getter and Setter  

### Welcome more tests, [pull requests](https://github.com/cxr29/scrud/pulls), [issues](https://github.com/cxr29/scrud/issues)...
