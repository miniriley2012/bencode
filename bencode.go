// Package bencode contains functions to bencode and bdecode data.
package bencode

import (
	"bytes"
	"errors"
	"reflect"
	"sort"
	"strconv"
	"strings"
)

// Marshaler is the interface implemented by types that
// can marshal themselves in to valid bencoded data.
type Marshaler interface {
	MarshalBencode() ([]byte, error)
}

// Unmarshaler is the interface implemented by types that
// can be unmarshalled from a byte slice of bencoded data.
type Unmarshaler interface {
	UnmarshalBencode([]byte) (int, error)
}

// Marshal returns the bencoded form of v.
func Marshal(v interface{}) ([]byte, error) {
	value := reflect.ValueOf(v)

	if v, ok := value.Interface().(Marshaler); ok {
		return v.MarshalBencode()
	}

	switch value.Kind() {
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return marshalInt(int(value.Int())), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return marshalInt(int(value.Uint())), nil
	case reflect.String:
		return marshalString(value.String()), nil
	case reflect.Array, reflect.Slice:
		return marshalList(value)
	case reflect.Map:
		return marshalMap(value)
	case reflect.Struct:
		return marshalStruct(value)
	case reflect.Ptr:
		return Marshal(value.Elem().Interface())
	}

	return nil, errors.New("unsupported type " + value.Type().String())
}

// marshalInt returns the bencoded form of i.
// No error is returned as this function cannot error.
func marshalInt(i int) []byte { return []byte("i" + strconv.Itoa(i) + "e") }

// marshalString returns the bencoded form of str.
// No error is returned as this function cannot error.
func marshalString(str string) []byte { return []byte(strconv.Itoa(len(str)) + ":" + str) }

// marshalList returns the bencoded form of a slice or array.
func marshalList(value reflect.Value) ([]byte, error) {
	if value.Type().Kind() == reflect.Slice && value.Type().Elem().Kind() == reflect.Uint8 {
		return marshalString(string(value.Bytes())), nil
	}

	var buf bytes.Buffer
	buf.WriteByte('l')
	for i := 0; i < value.Len(); i++ {
		b, err := Marshal(value.Index(i).Interface())
		buf.Write(b)
		if err != nil {
			buf.WriteByte('e')
			return buf.Bytes(), err
		}
	}
	buf.WriteByte('e')
	return buf.Bytes(), nil
}

// marshalMap returns the bencoded form of map.
func marshalMap(value reflect.Value) ([]byte, error) {
	if value.Type().Key().Kind() != reflect.String {
		return nil, errors.New("key type must be string")
	}

	dictionary := map[string]interface{}{}
	var keys []string
	for mapRange := value.MapRange(); mapRange.Next(); {
		keys = append(keys, mapRange.Key().String())
		dictionary[mapRange.Key().String()] = mapRange.Value().Interface()
	}
	sort.Strings(keys)

	var buf bytes.Buffer
	buf.WriteByte('d')

	for _, key := range keys {
		b, err := Marshal(dictionary[key])
		if err != nil {
			buf.WriteByte('e')
			return nil, err
		} else if len(b) > 0 {
			buf.Write(marshalString(key))
			buf.Write(b)
		}
	}

	buf.WriteByte('e')
	return buf.Bytes(), nil
}

// marshalStruct returns the bencoded form of a struct as a dictionary.
func marshalStruct(value reflect.Value) ([]byte, error) {
	var buf bytes.Buffer

	m := map[string]interface{}{}
	for i := 0; i < value.NumField(); i++ {
		f := value.Field(i)
		field := value.Type().Field(i)

		if field.PkgPath != "" {
			continue
		}

		var name string
		var omit, omitEmpty bool
		if v, ok := field.Tag.Lookup("bencode"); ok {
			name, omit, omitEmpty = parseTag(v)
			if omit || (omitEmpty && f.IsZero()) {
				continue
			}
		} else {
			name = field.Name
		}

		m[name] = f.Interface()
	}

	b, err := marshalMap(reflect.ValueOf(m))
	if err != nil {
		return nil, err
	}

	buf.Write(b)

	return buf.Bytes(), nil
}

func parseTag(tag string) (name string, omit bool, omitEmpty bool) {
	split := strings.SplitN(tag, ",", 2)
	return split[0], len(split) == 1 && split[0] == "-", len(split) > 1 && split[1] == "omitempty"
}

// Unmarshal sets v to be the decoded result of data. v must be a pointer.
func Unmarshal(data []byte, v interface{}) (int, error) {
	value := reflect.ValueOf(v)
	if !value.IsValid() {
		v = new(interface{})
	} else if value.Kind() != reflect.Ptr {
		return 0, errors.New("value is not a pointer")
	}

	switch data[0] {
	case 'i':
		var i int64
		n, err := unmarshalInt(data, &i)
		reflect.ValueOf(v).Elem().SetInt(i)
		return n, err
	case 'l':
		return unmarshalList(data, v)
	case 'd':
		return unmarshalDictionary(data, v)
	default:
		return unmarshalString(data, v.(*string))
	}
}

func unmarshalInt(data []byte, v *int64) (int, error) {
	var s string
	i := 1
	for ; data[i] != 'e'; i++ {
		s += string(data[i])
	}
	var err error
	*v, err = strconv.ParseInt(s, 10, 64)
	return i, err
}

func unmarshalString(data []byte, s *string) (i int, err error) {
	var n string
	for ; data[i] != ':'; i++ {
		n += string(data[i])
	}
	i++
	size, err := strconv.Atoi(n)
	if err != nil {
		return 0, err
	}
	*s = string(data[i : i+size])
	i += size
	return i, nil
}

func unmarshalList(data []byte, l interface{}) (int, error) {
	v := reflect.ValueOf(l)

	i := 1
	for data[i] != 'e' {
		var value interface{}
		switch data[i] {
		case 'i':
			value = new(int)
		case 'l':
			value = new([]interface{})
		case 'd':
			value = &map[string]interface{}{}
		default:
			value = new(string)
		}
		if n, err := Unmarshal(data[i:], value); err != nil {
			return i, err
		} else {
			i += n
		}

		v.Elem().Set(reflect.Append(v.Elem(), reflect.ValueOf(value).Elem()))

		if _, ok := value.(*string); !ok {
			i++
		}
	}
	return i, nil
}

func unmarshalDictionary(data []byte, d interface{}) (int, error) {
	v := reflect.ValueOf(d)
	i := 1
	for i < len(data) && data[i] != 'e' {
		var key string
		if n, err := unmarshalString(data[i:], &key); err != nil {
			return i, err
		} else {
			i += n
		}

		if v.Elem().Kind() == reflect.Struct {
			if f, ok := fieldWithNameOrTag(v.Elem(), key); ok {
				if u, ok := addrIfNotPtr(f).Interface().(Unmarshaler); ok {
					if n, err := u.UnmarshalBencode(data[i:]); err != nil {
						return i, err
					} else {
						i += n
					}
					if data[i] == 'e' {
						i++
					}
					continue
				}
			}
		}

		var value interface{}
		switch data[i] {
		case 'i':
			value = new(int)
		case 'l':
			value = new([]interface{})
		case 'd':
			value = &map[string]interface{}{}
		default:
			value = new(string)
		}
		if n, err := Unmarshal(data[i:], value); err != nil {
			return i, err
		} else {
			i += n
		}

		switch v.Elem().Kind() {
		case reflect.Map:
			v.Elem().SetMapIndex(reflect.ValueOf(key), reflect.ValueOf(value).Elem())
		case reflect.Struct:
			if f, ok := fieldWithNameOrTag(v.Elem(), key); ok {
				if f.Kind() == reflect.Slice {
					fillSlice(f, reflect.ValueOf(value).Elem())
				} else if f.Kind() == reflect.Array {
					fillArray(f, reflect.ValueOf(value).Elem())
				} else if f.Kind() == reflect.Struct {
					fillStruct(f, *value.(*map[string]interface{}))
				} else if f.Kind() == reflect.Ptr {
					f.Set(reflect.ValueOf(value).Convert(f.Type()))
				} else {
					f.Set(reflect.ValueOf(value).Elem().Convert(f.Type()))
				}
			}
		default:
			return i, errors.New("type is not a struct or map")
		}

		if _, ok := value.(*string); !ok {
			i++
		}
	}
	return i, nil
}
