package gqbd

import (
	"strconv"
	"strings"
)

// BuildConnectionString creates a database connection string based on the
// database type and configuration: a DSN for sql.Open(), a redis:// URL for
// Redis clients (e.g. redis.ParseURL in go-redis), a mongodb:// URI for
// options.Client().ApplyURI, a clickhouse:// DSN for clickhouse-go, or an
// oracle:// URL for go-ora.
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
	case MongoDB:
		return buildMongoDBConnectionString(config)
	case ClickHouse:
		return buildClickHouseConnectionString(config)
	case Oracle:
		return buildOracleConnectionString(config)
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

// buildClickHouseConnectionString returns the clickhouse-go v2 DSN
// clickhouse://[user:password@]host[:port]/[db][?secure=true].
func buildClickHouseConnectionString(config DBConfig) string {
	var b strings.Builder
	b.WriteString("clickhouse://")
	writeURLAuthority(&b, config, "localhost")
	b.WriteByte('/')
	b.WriteString(percentEncode(config.DBName))
	if config.SSLMode != "" && config.SSLMode != "disable" {
		b.WriteString("?secure=true")
	}
	return b.String()
}

// buildOracleConnectionString returns the go-ora URL
// oracle://[user:password@]host[:port]/service[?ssl=true], where DBName is the
// service name.
func buildOracleConnectionString(config DBConfig) string {
	var b strings.Builder
	b.WriteString("oracle://")
	writeURLAuthority(&b, config, "localhost")
	b.WriteByte('/')
	b.WriteString(percentEncode(config.DBName))
	if config.SSLMode != "" && config.SSLMode != "disable" {
		b.WriteString("?ssl=true")
	}
	return b.String()
}

// buildMongoDBConnectionString returns mongodb://[user:password@]host[:port]/[db][?tls=true].
func buildMongoDBConnectionString(config DBConfig) string {
	var b strings.Builder
	b.WriteString("mongodb://")
	writeURLAuthority(&b, config, "localhost")
	b.WriteByte('/')
	b.WriteString(percentEncode(config.DBName))
	if config.SSLMode != "" && config.SSLMode != "disable" {
		b.WriteString("?tls=true")
	}
	return b.String()
}

// buildRedisConnectionString returns redis://[user:password@]host[:port][/db],
// or rediss:// when SSLMode is set to anything other than "disable".
// It is written by hand rather than with net/url so the package imports no
// networking code.
func buildRedisConnectionString(config DBConfig) string {
	var b strings.Builder
	if config.SSLMode != "" && config.SSLMode != "disable" {
		b.WriteString("rediss://")
	} else {
		b.WriteString("redis://")
	}
	writeURLAuthority(&b, config, "localhost")
	if config.DBName != "" {
		b.WriteByte('/')
		b.WriteString(percentEncode(config.DBName))
	}
	return b.String()
}

// writeURLAuthority writes [user[:password]@]host[:port], percent-encoding the
// credentials and bracketing IPv6 hosts.
func writeURLAuthority(b *strings.Builder, config DBConfig, defaultHost string) {
	if config.User != "" || config.Password != "" {
		b.WriteString(percentEncode(config.User))
		if config.Password != "" {
			b.WriteByte(':')
			b.WriteString(percentEncode(config.Password))
		}
		b.WriteByte('@')
	}
	host := config.Host
	if host == "" {
		host = defaultHost
	}
	if strings.IndexByte(host, ':') >= 0 && host[0] != '[' {
		host = "[" + host + "]" // IPv6 literal
	}
	b.WriteString(host)
	if config.Port > 0 {
		b.WriteByte(':')
		b.WriteString(strconv.Itoa(config.Port))
	}
}

// percentEncode escapes every byte except RFC 3986 unreserved characters.
func percentEncode(s string) string {
	const hex = "0123456789ABCDEF"
	var b strings.Builder
	for i := 0; i < len(s); i++ {
		c := s[i]
		if c == '-' || c == '.' || c == '_' || c == '~' || isPlainIdent(s[i:i+1]) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
	return b.String()
}
