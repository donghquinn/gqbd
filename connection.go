package gqbd

import (
	"net"
	"net/url"
	"strconv"
	"strings"
)

// BuildConnectionString creates a database connection string based on the
// database type and configuration: a DSN for sql.Open(), or a redis:// URL
// for Redis clients (e.g. redis.ParseURL in go-redis).
//
// Examples:
//
//	config := gqbd.DBConfig{
//	    Host: "localhost", Port: 5432, User: "postgres",
//	    Password: "password", DBName: "mydb", SSLMode: "disable"
//	}
//	dsn := gqbd.BuildConnectionString(gqbd.PostgreSQL, config)
func BuildConnectionString(dbType DBType, config DBConfig) string {
	switch dbType {
	case PostgreSQL:
		return buildPostgreSQLConnectionString(config)
	case SQLite:
		return buildSQLiteConnectionString(config)
	case Redis:
		return buildRedisConnectionString(config)
	default:
		return buildMySQLConnectionString(config)
	}
}

func buildPostgreSQLConnectionString(config DBConfig) string {
	var parts []string

	if config.Host != "" {
		parts = append(parts, "host="+pgValue(config.Host))
	}
	if config.Port > 0 {
		parts = append(parts, "port="+strconv.Itoa(config.Port))
	}
	if config.User != "" {
		parts = append(parts, "user="+pgValue(config.User))
	}
	if config.Password != "" {
		parts = append(parts, "password="+pgValue(config.Password))
	}
	if config.DBName != "" {
		parts = append(parts, "dbname="+pgValue(config.DBName))
	}
	if config.SSLMode != "" {
		parts = append(parts, "sslmode="+pgValue(config.SSLMode))
	} else {
		parts = append(parts, "sslmode=disable")
	}

	return strings.Join(parts, " ")
}

// pgValue quotes a key=value connection string value when it contains
// characters that would otherwise end or corrupt the value.
func pgValue(v string) string {
	if !strings.ContainsAny(v, ` '\`) {
		return v
	}
	v = strings.ReplaceAll(v, `\`, `\\`)
	v = strings.ReplaceAll(v, `'`, `\'`)
	return "'" + v + "'"
}

func buildMySQLConnectionString(config DBConfig) string {
	var dsn strings.Builder

	if config.User != "" {
		dsn.WriteString(config.User)
	}
	if config.Password != "" {
		dsn.WriteString(":" + config.Password)
	}
	dsn.WriteString("@")

	if config.Host != "" {
		dsn.WriteString("tcp(" + config.Host)
		if config.Port > 0 {
			dsn.WriteString(":" + strconv.Itoa(config.Port))
		}
		dsn.WriteString(")")
	}

	if config.DBName != "" {
		dsn.WriteString("/" + config.DBName)
	}

	dsn.WriteString("?")
	if config.Charset != "" {
		dsn.WriteString("charset=" + config.Charset + "&")
	}
	dsn.WriteString("parseTime=True&loc=Local")

	return dsn.String()
}

func buildSQLiteConnectionString(config DBConfig) string {
	if config.FilePath != "" {
		return config.FilePath
	}
	return ":memory:"
}

// buildRedisConnectionString returns redis://[user:password@]host[:port][/db],
// or rediss:// when SSLMode is set to anything other than "disable".
func buildRedisConnectionString(config DBConfig) string {
	u := url.URL{Scheme: "redis", Host: config.Host}
	if config.SSLMode != "" && config.SSLMode != "disable" {
		u.Scheme = "rediss"
	}
	if u.Host == "" {
		u.Host = "localhost"
	}
	if config.Port > 0 {
		u.Host = net.JoinHostPort(u.Host, strconv.Itoa(config.Port))
	}
	if config.Password != "" {
		u.User = url.UserPassword(config.User, config.Password)
	} else if config.User != "" {
		u.User = url.User(config.User)
	}
	if config.DBName != "" {
		u.Path = "/" + config.DBName
	}
	return u.String()
}
