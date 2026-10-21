package gqbd

import (
	"fmt"
	"math"
	"reflect"
	"sort"
	"strconv"
	"strings"
	"time"
)

// D is an ordered document, used where MongoDB depends on key order, such as
// sort specifications. It mirrors bson.D from the official driver and
// implements its bson.Marshaler interface (MarshalBSON), so a D can be passed
// anywhere the driver accepts a document without gqbd importing the driver.
type D []E

// E is a single key/value element of a D.
type E struct {
	Key   string
	Value interface{}
}

// MarshalBSON encodes d as a BSON document. It supports nil, bool, strings,
// integers, floats, time.Time, []byte, D, string-keyed maps, slices and arrays,
// [12]byte (as an ObjectID), and values implementing MarshalBSON or the
// driver's MarshalBSONValue.
func (d D) MarshalBSON() ([]byte, error) {
	return appendBSONDocument(make([]byte, 0, 16+16*len(d)), d)
}

const (
	bsonDouble   byte = 0x01
	bsonString   byte = 0x02
	bsonDocument byte = 0x03
	bsonArray    byte = 0x04
	bsonBinary   byte = 0x05
	bsonObjectID byte = 0x07
	bsonBool     byte = 0x08
	bsonDateTime byte = 0x09
	bsonNull     byte = 0x0A
	bsonInt32    byte = 0x10
	bsonInt64    byte = 0x12
)

// bsonMarshaler and bsonValueMarshaler match the driver's bson.Marshaler and
// (v2) bson.ValueMarshaler interfaces.
type bsonMarshaler interface {
	MarshalBSON() ([]byte, error)
}

type bsonValueMarshaler interface {
	MarshalBSONValue() (byte, []byte, error)
}

func appendBSONDocument(dst []byte, d D) ([]byte, error) {
	start := len(dst)
	dst = append(dst, 0, 0, 0, 0)
	for _, e := range d {
		var err error
		if dst, err = appendBSONElement(dst, e.Key, e.Value); err != nil {
			return nil, err
		}
	}
	return closeBSONDocument(dst, start), nil
}

func closeBSONDocument(dst []byte, start int) []byte {
	dst = append(dst, 0)
	putUint32(dst[start:], uint32(len(dst)-start))
	return dst
}

func appendBSONElement(dst []byte, key string, v interface{}) ([]byte, error) {
	if strings.IndexByte(key, 0) >= 0 {
		return nil, fmt.Errorf("BSON key %q contains a NUL byte", key)
	}
	typePos := len(dst)
	dst = append(dst, 0)
	dst = append(dst, key...)
	dst = append(dst, 0)
	dst, typ, err := appendBSONValue(dst, v)
	if err != nil {
		return nil, fmt.Errorf("key %q: %w", key, err)
	}
	dst[typePos] = typ
	return dst, nil
}

func appendBSONValue(dst []byte, v interface{}) ([]byte, byte, error) {
	switch x := v.(type) {
	case nil:
		return dst, bsonNull, nil
	case D:
		dst, err := appendBSONDocument(dst, x)
		return dst, bsonDocument, err
	case bsonValueMarshaler:
		typ, data, err := x.MarshalBSONValue()
		return append(dst, data...), typ, err
	case bsonMarshaler:
		data, err := x.MarshalBSON()
		return append(dst, data...), bsonDocument, err
	case bool:
		if x {
			return append(dst, 1), bsonBool, nil
		}
		return append(dst, 0), bsonBool, nil
	case string:
		dst = appendInt32(dst, int32(len(x)+1))
		dst = append(dst, x...)
		return append(dst, 0), bsonString, nil
	case int:
		return appendBSONInt(dst, int64(x))
	case int8:
		return appendInt32(dst, int32(x)), bsonInt32, nil
	case int16:
		return appendInt32(dst, int32(x)), bsonInt32, nil
	case int32:
		return appendInt32(dst, x), bsonInt32, nil
	case int64:
		return appendInt64(dst, x), bsonInt64, nil
	case uint8:
		return appendInt32(dst, int32(x)), bsonInt32, nil
	case uint16:
		return appendInt32(dst, int32(x)), bsonInt32, nil
	case uint32:
		return appendInt64(dst, int64(x)), bsonInt64, nil
	case uint:
		return appendBSONUint(dst, uint64(x))
	case uint64:
		return appendBSONUint(dst, x)
	case float32:
		return appendInt64(dst, int64(math.Float64bits(float64(x)))), bsonDouble, nil
	case float64:
		return appendInt64(dst, int64(math.Float64bits(x))), bsonDouble, nil
	case time.Time:
		ms := x.Unix()*1000 + int64(x.Nanosecond())/int64(time.Millisecond)
		return appendInt64(dst, ms), bsonDateTime, nil
	case []byte:
		dst = appendInt32(dst, int32(len(x)))
		dst = append(dst, 0) // generic binary subtype
		return append(dst, x...), bsonBinary, nil
	case map[string]interface{}:
		return appendBSONMap(dst, reflect.ValueOf(x))
	case []interface{}:
		return appendBSONArray(dst, reflect.ValueOf(x))
	}
	return appendBSONReflect(dst, reflect.ValueOf(v))
}

// appendBSONReflect handles named types (bson.M, bson.A, bson.D, ObjectID, ...)
// by their underlying kind.
func appendBSONReflect(dst []byte, rv reflect.Value) ([]byte, byte, error) {
	if rv.IsValid() && isDriverBSONType(rv.Type()) {
		switch rv.Type().Name() {
		case "DateTime":
			return appendInt64(dst, rv.Int()), bsonDateTime, nil
		case "ObjectID", "D", "M", "A":
			// Encoded by kind below.
		default:
			return nil, 0, fmt.Errorf("unsupported BSON value of type %s inside gqbd.D; put it in a filter or map, which the driver encodes", rv.Type())
		}
	}
	switch rv.Kind() {
	case reflect.Ptr, reflect.Interface:
		if rv.IsNil() {
			return dst, bsonNull, nil
		}
		return appendBSONValue(dst, rv.Elem().Interface())
	case reflect.Bool:
		return appendBSONValue(dst, rv.Bool())
	case reflect.String:
		return appendBSONValue(dst, rv.String())
	case reflect.Int:
		return appendBSONInt(dst, rv.Int())
	case reflect.Int8, reflect.Int16, reflect.Int32:
		return appendInt32(dst, int32(rv.Int())), bsonInt32, nil
	case reflect.Int64:
		return appendInt64(dst, rv.Int()), bsonInt64, nil
	case reflect.Uint8, reflect.Uint16:
		return appendInt32(dst, int32(rv.Uint())), bsonInt32, nil
	case reflect.Uint, reflect.Uint32, reflect.Uint64:
		return appendBSONUint(dst, rv.Uint())
	case reflect.Float32, reflect.Float64:
		return appendBSONValue(dst, rv.Float())
	case reflect.Map:
		if rv.Type().Key().Kind() != reflect.String {
			return nil, 0, fmt.Errorf("unsupported map key type %s", rv.Type().Key())
		}
		return appendBSONMap(dst, rv)
	case reflect.Array:
		if rv.Len() == 12 && rv.Type().Elem().Kind() == reflect.Uint8 {
			for i := 0; i < 12; i++ {
				dst = append(dst, byte(rv.Index(i).Uint()))
			}
			return dst, bsonObjectID, nil
		}
		return appendBSONArray(dst, rv)
	case reflect.Slice:
		if rv.Type().Elem().Kind() == reflect.Uint8 {
			return appendBSONValue(dst, rv.Bytes())
		}
		if isKeyValueSlice(rv.Type()) {
			return appendBSONKeyValues(dst, rv)
		}
		return appendBSONArray(dst, rv)
	}
	if !rv.IsValid() {
		return dst, bsonNull, nil
	}
	return nil, 0, fmt.Errorf("unsupported BSON value of type %s", rv.Type())
}

func appendBSONInt(dst []byte, i int64) ([]byte, byte, error) {
	if i >= math.MinInt32 && i <= math.MaxInt32 {
		return appendInt32(dst, int32(i)), bsonInt32, nil
	}
	return appendInt64(dst, i), bsonInt64, nil
}

func appendBSONUint(dst []byte, u uint64) ([]byte, byte, error) {
	if u > math.MaxInt64 {
		return nil, 0, fmt.Errorf("%d overflows int64", u)
	}
	return appendInt64(dst, int64(u)), bsonInt64, nil
}

// appendBSONMap encodes a string-keyed map with keys in sorted order.
func appendBSONMap(dst []byte, rv reflect.Value) ([]byte, byte, error) {
	keys := make([]string, 0, rv.Len())
	for _, k := range rv.MapKeys() {
		keys = append(keys, k.String())
	}
	sort.Strings(keys)
	start := len(dst)
	dst = append(dst, 0, 0, 0, 0)
	keyType := rv.Type().Key()
	for _, k := range keys {
		var err error
		dst, err = appendBSONElement(dst, k, rv.MapIndex(reflect.ValueOf(k).Convert(keyType)).Interface())
		if err != nil {
			return nil, 0, err
		}
	}
	return closeBSONDocument(dst, start), bsonDocument, nil
}

func appendBSONArray(dst []byte, rv reflect.Value) ([]byte, byte, error) {
	start := len(dst)
	dst = append(dst, 0, 0, 0, 0)
	for i := 0; i < rv.Len(); i++ {
		var err error
		if dst, err = appendBSONElement(dst, strconv.Itoa(i), rv.Index(i).Interface()); err != nil {
			return nil, 0, err
		}
	}
	return closeBSONDocument(dst, start), bsonArray, nil
}

// isDriverBSONType reports whether t is declared in the official driver's bson
// packages (v1 bson/primitive or v2 bson), whose types need their own BSON type codes.
func isDriverBSONType(t reflect.Type) bool {
	pkg := t.PkgPath()
	return strings.HasPrefix(pkg, "go.mongodb.org/mongo-driver") && strings.HasSuffix(pkg, "bson") ||
		strings.HasPrefix(pkg, "go.mongodb.org/mongo-driver") && strings.HasSuffix(pkg, "bson/primitive")
}

// isKeyValueSlice reports whether t is a slice of struct{Key string; Value interface{}},
// the shape of the driver's bson.D.
func isKeyValueSlice(t reflect.Type) bool {
	e := t.Elem()
	if e.Kind() != reflect.Struct || e.NumField() != 2 {
		return false
	}
	k, v := e.Field(0), e.Field(1)
	return k.Name == "Key" && k.Type.Kind() == reflect.String &&
		v.Name == "Value" && v.Type.Kind() == reflect.Interface
}

func appendBSONKeyValues(dst []byte, rv reflect.Value) ([]byte, byte, error) {
	start := len(dst)
	dst = append(dst, 0, 0, 0, 0)
	for i := 0; i < rv.Len(); i++ {
		el := rv.Index(i)
		var err error
		if dst, err = appendBSONElement(dst, el.Field(0).String(), el.Field(1).Interface()); err != nil {
			return nil, 0, err
		}
	}
	return closeBSONDocument(dst, start), bsonDocument, nil
}

func appendInt32(dst []byte, v int32) []byte {
	u := uint32(v)
	return append(dst, byte(u), byte(u>>8), byte(u>>16), byte(u>>24))
}

func appendInt64(dst []byte, v int64) []byte {
	u := uint64(v)
	return append(dst, byte(u), byte(u>>8), byte(u>>16), byte(u>>24),
		byte(u>>32), byte(u>>40), byte(u>>48), byte(u>>56))
}

func putUint32(b []byte, v uint32) {
	b[0], b[1], b[2], b[3] = byte(v), byte(v>>8), byte(v>>16), byte(v>>24)
}
