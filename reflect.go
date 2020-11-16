package bencode

import "reflect"

func indirect(v reflect.Value) reflect.Value {
	if v.Kind() == reflect.Ptr || v.Kind() == reflect.Interface {
		return v.Elem()
	}
	return v
}

func fieldWithNameOrTag(s reflect.Value, name string) (field reflect.Value, ok bool) {
	for i := 0; i < s.NumField(); i++ {
		if n, ok := s.Type().Field(i).Tag.Lookup("bencode"); ok && name == n || s.Type().Field(i).Name == name {
			return s.Field(i), true
		}
	}
	return
}

func fillStruct(s reflect.Value, m map[string]interface{}) {
	for k, v := range m {
		if f, ok := fieldWithNameOrTag(s, k); ok {
			if f.Kind() == reflect.Array || f.Kind() == reflect.Slice {
				value := reflect.ValueOf(m[k])
				for i := 0; i < value.Len(); i++ {
					if value.Kind() == reflect.Struct {
						fillStruct(f, m[k].(map[string]interface{}))
					} else if value.Kind() == reflect.Slice {
						fillSlice(f, value)
					} else {
						f.Set(reflect.Append(f, value.Index(i).Convert(f.Type().Elem())))
					}
				}
			} else if f.Kind() == reflect.Struct {
				fillStruct(s, m[k].(map[string]interface{}))
			} else {
				f.Set(reflect.ValueOf(v).Convert(f.Type()))
			}
		}
	}
}

func fillSlice(dst reflect.Value, src reflect.Value) {
	for i := 0; i < src.Len(); i++ {
		v := indirect(src.Index(i))
		if v.Kind() == reflect.Array || v.Kind() == reflect.Slice {
			val := reflect.New(reflect.SliceOf(dst.Type().Elem().Elem())).Elem()
			fillSlice(val, v)
			dst.Set(reflect.Append(dst, val))
		} else if dst.Type().Elem().Kind() == reflect.Struct {
			val := reflect.New(dst.Type().Elem()).Elem()
			fillStruct(val, v.Interface().(map[string]interface{}))
			dst.Set(reflect.Append(dst, val))
		} else {
			dst.Set(reflect.Append(dst, v))
		}
	}
}

func addrIfNotPtr(v reflect.Value) reflect.Value {
	if v.Kind() != reflect.Ptr {
		return v.Addr()
	}
	return v
}
