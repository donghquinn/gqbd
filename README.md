# go-query-builder

## Introduction
* It's Go Query Building package for dynamic queries.
* You can make Query String simillar way with QueryDsl of Java.
* It creates prepared statements
    * SQL Injection Concerned.
* It support Mariadb/Mysql and PostgreSQL so far
    * If you have any other one, please let me know

## Installation

```zsh
go get git@github.com:donghquinn/go-query-builder.git
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

```go
package example

import 	gqbd "github.com/donghquinn/go-query-builder"

func example() {
    dbCon, conErr := database.PostgresConnection()

    /*
        Logics
    */

    // Arguments: DB Type, Table Name, Columns...
    qb := gqbd.NewQueryBuilder("postgres", "example_table", "exam_id", "exam_name").
        Where("exam_name = ?", "data")
    
    queryString, args, buildErr := qb.Build()

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

### Mysql / Mariadb
* It uses ? for prepared statment

```go
package example

import 	gqbd "github.com/donghquinn/go-query-builder"

func example() {
    dbCon, conErr := database.MariadbConnection()

    /*
        Logics
    */

    // Arguments: DB Type, Table Name, Columns...
    qb := gqbd.NewQueryBuilder("mariadb", "example_table", "exam_id", "exam_name").
        Where("exam_name = ?", "data")
    
    queryString, args, buildErr := qb.Build()

    /*
        @@ Query String Result @@
        SELECT exam_id, exam_name
        FROM example_table
        WHERE exam_name = ?

        @@ Query Arguments @@
        "data"
    */
    queryResult, queryErr := dbCon.QueryRows(queryString, args)
     
    /*
        Query Result Error Handling
    */
}

```
