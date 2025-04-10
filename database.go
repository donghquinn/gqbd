package gqbd

import (
	"database/sql"
	"fmt"
	"log"
	"time"
)

type DBConfig struct {
	UserName     string
	Password     string
	Host         string
	Port         int
	Database     string
	SslMode      string
	MaxLifeTime  time.Duration
	MaxIdleConns int
	MaxOpenConns int
}

type DataBaseConnector struct {
	*sql.DB
}

func InitConnection(dbType DBType, cfg DBConfig) (*DataBaseConnector, error) {
	switch dbType {
	case MariaDB:
		return InitMariadbConnection(cfg)
	case Mysql:
		return InitMariadbConnection(cfg)
	case PostgreSQL:
		return InitPostgresConnection(cfg)
	default:
		return nil, fmt.Errorf("unsupported DB type: %s", dbType)
	}
}

/*
Common Query Method for Single Row
*/
func (connect *DataBaseConnector) QueryBuilderOneRow(queryString string, args []interface{}) (*sql.Row, error) {
	result := connect.QueryRow(queryString, args...)

	if result.Err() != nil {
		log.Printf("[QUERY] Query Error: %v\n", result.Err())

		return nil, result.Err()
	}

	defer connect.Close()

	return result, nil
}

/*
Common Query Method for Multiple Rows
*/
func (connect *DataBaseConnector) QueryBuilderRows(queryString string, args []interface{}) (*sql.Rows, error) {
	result, err := connect.Query(queryString, args...)

	if err != nil {
		log.Printf("[QUERY] Query Error: %v\n", err)

		return nil, err
	}

	defer connect.Close()

	return result, nil
}
