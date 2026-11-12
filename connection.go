package gqbd

import (
	"strconv"
	"strings"
)

// BuildConnectionString creates a database connection string based on the
// database type and configuration: a DSN for sql.Open(), a redis:// URL for
// Redis clients (e.g. redis.ParseURL in go-redis), a mongodb:// URI for
// options.Client().ApplyURI, a clickhouse:// DSN for clickhouse-go, an
// oracle:// URL for go-ora, or an http(s):// address for go-elasticsearch,
// the Prometheus HTTP API client or the Qdrant REST API.
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
	case Elasticsearch, Prometheus:
		return buildHTTPConnectionString(config)
	case Qdrant:
		// Qdrant authenticates with an api-key header, not basic auth, so the
		// credentials are left out of the address.
		config.User, config.Password = "", ""
		return buildHTTPConnectionString(config)
	default:
		return buildMySQLConnectionString(config)
	}
}

func buildPostgreSQLConnectionString(config DBConfig) string {
	var b strings.Builder
	b.Grow(64 + len(config.Host) + len(config.User) + len(config.Password) + len(config.DBName) + len(config.SSLMode))

	if config.Host != "" {
		writePgParam(&b, "host=", config.Host)
	}
	if config.Port > 0 {
		if b.Len() > 0 {
			b.WriteByte(' ')
		}
		b.WriteString("port=")
		writePort(&b, config.Port)
	}
	if config.User != "" {
		writePgParam(&b, "user=", config.User)
	}
	if config.Password != "" {
		writePgParam(&b, "password=", config.Password)
	}
	if config.DBName != "" {
		writePgParam(&b, "dbname=", config.DBName)
	}
	if config.SSLMode != "" {
		writePgParam(&b, "sslmode=", config.SSLMode)
	} else {
		writePgParam(&b, "sslmode=", "disable")
	}

	return b.String()
}

// writePgParam writes a space-separated key=value pair, quoting the value
// when it contains characters that would otherwise end or corrupt it.
func writePgParam(b *strings.Builder, key, v string) {
	if b.Len() > 0 {
		b.WriteByte(' ')
	}
	b.WriteString(key)
	if !strings.ContainsAny(v, ` '\`) {
		b.WriteString(v)
		return
	}
	b.WriteByte('\'')
	for i := 0; i < len(v); i++ {
		if v[i] == '\\' || v[i] == '\'' {
			b.WriteByte('\\')
		}
		b.WriteByte(v[i])
	}
	b.WriteByte('\'')
}

func buildMySQLConnectionString(config DBConfig) string {
	var dsn strings.Builder
	dsn.Grow(64 + len(config.User) + len(config.Password) + len(config.Host) + len(config.DBName) + len(config.Charset))

	if config.User != "" {
		dsn.WriteString(config.User)
	}
	if config.Password != "" {
		dsn.WriteByte(':')
		dsn.WriteString(config.Password)
	}
	dsn.WriteByte('@')

	if config.Host != "" {
		dsn.WriteString("tcp(")
		dsn.WriteString(config.Host)
		if config.Port > 0 {
			dsn.WriteByte(':')
			writePort(&dsn, config.Port)
		}
		dsn.WriteByte(')')
	}

	if config.DBName != "" {
		dsn.WriteByte('/')
		dsn.WriteString(config.DBName)
	}

	dsn.WriteByte('?')
	if config.Charset != "" {
		dsn.WriteString("charset=")
		dsn.WriteString(config.Charset)
		dsn.WriteByte('&')
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
	b.Grow(urlLen(config))
	b.WriteString("clickhouse://")
	writeURLAuthority(&b, config, "localhost")
	b.WriteByte('/')
	writePercentEncoded(&b, config.DBName)
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
	b.Grow(urlLen(config))
	b.WriteString("oracle://")
	writeURLAuthority(&b, config, "localhost")
	b.WriteByte('/')
	writePercentEncoded(&b, config.DBName)
	if config.SSLMode != "" && config.SSLMode != "disable" {
		b.WriteString("?ssl=true")
	}
	return b.String()
}

// buildHTTPConnectionString returns http://[user:password@]host[:port], or
// https:// when SSLMode is set to anything other than "disable". go-elasticsearch
// and net/http (used by the Prometheus client) send the credentials as basic auth.
func buildHTTPConnectionString(config DBConfig) string {
	var b strings.Builder
	b.Grow(urlLen(config))
	if config.SSLMode != "" && config.SSLMode != "disable" {
		b.WriteString("https://")
	} else {
		b.WriteString("http://")
	}
	writeURLAuthority(&b, config, "localhost")
	return b.String()
}

// buildMongoDBConnectionString returns mongodb://[user:password@]host[:port]/[db][?tls=true].
func buildMongoDBConnectionString(config DBConfig) string {
	var b strings.Builder
	b.Grow(urlLen(config))
	b.WriteString("mongodb://")
	writeURLAuthority(&b, config, "localhost")
	b.WriteByte('/')
	writePercentEncoded(&b, config.DBName)
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
	b.Grow(urlLen(config))
	if config.SSLMode != "" && config.SSLMode != "disable" {
		b.WriteString("rediss://")
	} else {
		b.WriteString("redis://")
	}
	writeURLAuthority(&b, config, "localhost")
	if config.DBName != "" {
		b.WriteByte('/')
		writePercentEncoded(&b, config.DBName)
	}
	return b.String()
}

// writeURLAuthority writes [user[:password]@]host[:port], percent-encoding the
// credentials and bracketing IPv6 hosts.
func writeURLAuthority(b *strings.Builder, config DBConfig, defaultHost string) {
	if config.User != "" || config.Password != "" {
		writePercentEncoded(b, config.User)
		if config.Password != "" {
			b.WriteByte(':')
			writePercentEncoded(b, config.Password)
		}
		b.WriteByte('@')
	}
	host := config.Host
	if host == "" {
		host = defaultHost
	}
	if strings.IndexByte(host, ':') >= 0 && host[0] != '[' {
		b.WriteByte('[') // IPv6 literal
		b.WriteString(host)
		b.WriteByte(']')
	} else {
		b.WriteString(host)
	}
	if config.Port > 0 {
		b.WriteByte(':')
		writePort(b, config.Port)
	}
}

// writePort writes the port number without allocating an intermediate string.
func writePort(b *strings.Builder, port int) {
	var num [20]byte
	b.Write(strconv.AppendInt(num[:0], int64(port), 10))
}

// urlLen estimates the length of a URL-style connection string, for sizing
// its builder; percent-encoded credentials may make it longer.
func urlLen(config DBConfig) int {
	return 32 + len(config.User) + len(config.Password) + len(config.Host) + len(config.DBName)
}

// percentEncode escapes every byte except RFC 3986 unreserved characters.
func percentEncode(s string) string {
	if !needsPercentEncoding(s) {
		return s
	}
	var b strings.Builder
	b.Grow(len(s) + 16)
	writePercentEncoded(&b, s)
	return b.String()
}

func isUnreserved(c byte) bool {
	return c == '-' || c == '.' || c == '_' || c == '~' ||
		c >= 'a' && c <= 'z' || c >= 'A' && c <= 'Z' || c >= '0' && c <= '9'
}

func needsPercentEncoding(s string) bool {
	for i := 0; i < len(s); i++ {
		if !isUnreserved(s[i]) {
			return true
		}
	}
	return false
}

// writePercentEncoded writes percentEncode(s) to b.
func writePercentEncoded(b *strings.Builder, s string) {
	const hex = "0123456789ABCDEF"
	if !needsPercentEncoding(s) {
		b.WriteString(s)
		return
	}
	for i := 0; i < len(s); i++ {
		c := s[i]
		if isUnreserved(c) {
			b.WriteByte(c)
			continue
		}
		b.WriteByte('%')
		b.WriteByte(hex[c>>4])
		b.WriteByte(hex[c&15])
	}
}
