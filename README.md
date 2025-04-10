# go-query-builder

## Introduction
* It's Go Query Building package for dynamic queries.
* It creates prepared statements
    * SQL Injection Concerned.
* It support Mariadb/Mysql and PostgreSQL so far
    * If you have any other one, please let me know

## Installation

```zsh
go get github.com/donghquinn/gqbd
```


## Usage Examples

* First of all, create DB Connection.
*  You can give Database Type for creating prepared statments
    * You can use "postgres", "mariadb" and "mysql"
    * I'm opened to add more database types (Planning for sqlite3)
* It will retury Query string, arguments, and build error
    * build error is the error checking dbTypes
    * query string will contains ?(mariadb/mysql) or $N(postgres)

### Postgres
* It uses $N for prepared statment

##### SELECT

```go
package example

import 	"github.com/donghquinn/gqbd"

func example() {
    dbCon, conErr := database.PostgresConnection()

    /*
        Logics
    */

    // Arguments: DB Type, Table Name, Columns...
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "table_name", "col1").
		Where("col1 = ?", 100).
		OrderBy("col1", "ASC", nil).
		Limit(10).
		Offset(5)

	queryString, args, err := qb.Build()

    /*
        @@ Query String Result @@
        SELECT exam_id, exam_name
        FROM example_table
        WHERE exam_name = $1

        @@ Query Arguments @@
        "data"
    */
    queryResult, queryErr := dbCon.QueryRows(queryString, args)
     
    /*
        Query Result Error Handling
    */
}
```

#### Dynamic Where Expression

```go
package example

import 	"github.com/donghquinn/gqbd"

func example() {
    dbCon, conErr := database.PostgresConnection()

    /*
        Logics
    */

    // Arguments: DB Type, Table Name, Columns...
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "example_table e", "e.id", "e.name", "u.user").
		LeftJoin("user_table u", "u.user_id = e.id")

	if userName != "" {
		qb = qb.Where("u.user_name LIKE ?", "%"+userName+"%")
	}

	// title이 비어있지 않은 경우에만 조건 추가
	if title != "" {
		qb = qb.Where("e.name LIKE ?", "%"+title+"%")
	}
	// 상태 조건은 항상 추가
	qb = qb.Where("e.example_status = ?", "1")

	// 정렬, 오프셋, 제한 설정
	qb = qb.OrderBy(orderByColumn, "DESC", nil).
		Offset(offset).
		Limit(limit)

	queryString, args, err := qb.Build()

    /*
        @@ Query String Result @@
        SELECT exam_id, exam_name
        FROM example_table
        WHERE exam_name = $1

        @@ Query Arguments @@
        "data"
    */
    queryResult, queryErr := dbCon.QueryBuilderRows(queryString, args)
     
    /*
        Query Result Error Handling
    */
}
```

##### INSERT

```go
	data := map[string]interface{}{
		"col1": 200,
		"col2": "test",
	}
	qb := gqbd.BuildInsert(gqbd.PostgreSQL, "table_name").
		Values(data).
		Returning("col1")
	query, args, err := qb.Build()
```


##### UPDATE
 
```go
data := map[string]interface{}{
		"col1": 300,
		"col2": "update",
	}
	qb := gqbd.BuildUpdate(gqbd.PostgreSQL, "table_name").
		Set(data).
		Where("col1 = ?", 100)
	query, args, err := qb.Build()
```


### Mysql / Mariadb
* It uses ? for prepared statment

##### SELECT

```go
package example

import 	"github.com/donghquinn/gqbd"

func example() {
    dbCon, conErr := database.MariadbConnection()

    /*
        Logics
    */

    // Arguments: DB Type, Table Name, Columns...
	qb := gqbd.BuildSelect(gqbd.MariaDB, "table_name", "col1").
		Where("col1 = ?", 100).
		OrderBy("col1", "ASC", nil).
		Limit(10).
		Offset(5)

    queryString, args, buildErr := qb.Build()

    /*
        @@ Query String Result @@
        SELECT exam_id, exam_name
        FROM example_table
        WHERE exam_name = ?

        @@ Query Arguments @@
        "data"
    */
    queryResult, queryErr := dbCon.QueryBuilderRows(queryString, args)
     
    /*
        Query Result Error Handling
    */
}

```


#### Dynamic Where Expression

```go
package example

import 	"github.com/donghquinn/gqbd"

func example() {
    dbCon, conErr := database.PostgresConnection()

    /*
        Logics
    */

    // Arguments: DB Type, Table Name, Columns...
	qb := gqbd.BuildSelect(gqbd.PostgreSQL, "example_table e", "e.id", "e.name", "u.user").
		LeftJoin("user_table u", "u.user_id = e.id")

	if userName != "" {
		qb = qb.Where("u.user_name LIKE ?", "%"+userName+"%")
	}

	// title이 비어있지 않은 경우에만 조건 추가
	if title != "" {
		qb = qb.Where("e.name LIKE ?", "%"+title+"%")
	}
	// 상태 조건은 항상 추가
	qb = qb.Where("e.example_status = ?", "1")

	// 정렬, 오프셋, 제한 설정
	qb = qb.OrderBy(orderByColumn, "DESC", nil).
		Offset(offset).
		Limit(limit)

	queryString, args, err := qb.Build()

    /*
        @@ Query String Result @@
        SELECT exam_id, exam_name
        FROM example_table
        WHERE exam_name = $1

        @@ Query Arguments @@
        "data"
    */
    queryResult, queryErr := dbCon.QueryBuilderRows(queryString, args)
     
    /*
        Query Result Error Handling
    */
}
```

##### UPDATE

```go
data := map[string]interface{}{
		"col1": 300,
		"col2": "update",
	}
	qb := gqbd.BuildUpdate(gqbd.MariaDB, "table_name").
		Set(data).
		Where("col1 = ?", 100)
	query, args, err := qb.Build()
```

##### INSERT

```go
	data := map[string]interface{}{
		"col1": 200,
		"col2": "test",
	}
	qb := gqbd.BuildInsert(gqbd.MariaDB, "table_name").
		Values(data)

	query, args, err := qb.Build()

```
