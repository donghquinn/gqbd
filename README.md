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
    * You can use "postgres", "mariadb", "mysql"
* It will retury Query string, arguments, and build error
    * build error is the error checking dbTypes
    * query string will contains ?(mariadb/mysql) or $N(postgres)

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

    queryResult, queryErr := dbCon.QueryRows(queryString, args)
     
    /*
        Query Result Error Handling
    */
}

```
