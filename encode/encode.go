package encode

import (
	"encoding"
	"fmt"
	"reflect"
	"sync"

	"github.com/et-nik/binngo/binn"
)

type encoderFunc func(v reflect.Value) ([]byte, error)

var encoderCache sync.Map // map[reflect.Type]encoderFunc

var (
	textMarshalerType = reflect.TypeOf((*encoding.TextMarshaler)(nil)).Elem()
)

func marshal(v interface{}) ([]byte, error) {
	rv := reflect.ValueOf(v)

	if !rv.IsValid() {
		return nil, fmt.Errorf("Invalid value")
	}

	if !rv.IsValid() {
		return nil, fmt.Errorf("Invalid value")
	}

	enc := loadEncodeFunc(rv.Type())

	return enc(rv)
}

func loadEncodeFunc(t reflect.Type) encoderFunc {
	if fi, ok := encoderCache.Load(t); ok {
		return fi.(encoderFunc)
	}

	var (
		wg sync.WaitGroup
		f  encoderFunc
	)
	wg.Add(1)
	fi, loaded := encoderCache.LoadOrStore(t, encoderFunc(func(v reflect.Value) ([]byte, error) {
		wg.Wait()
		return f(v)
	}))
	if loaded {
		return fi.(encoderFunc)
	}

	f = newTypeEncoder(t)
	wg.Done()
	encoderCache.Store(t, f)
	return f
}

func newTypeEncoder(t reflect.Type) encoderFunc {
	if t.Implements(textMarshalerType) {
		return textMarshalerEncoder
	}

	switch t.Kind() {
	case reflect.Bool:
		return func(v reflect.Value) ([]byte, error) {
			if v.Bool() {
				return []byte{binn.True}, nil
			} else {
				return []byte{binn.False}, nil
			}
		}
	case reflect.Struct:
		return newStructEncoder(t)
	case reflect.Map:
		return newMapEncoder(t)
	case reflect.Interface:
		return func(v reflect.Value) ([]byte, error) {
			if v.IsNil() {
				return []byte{binn.Null}, nil
			}

			return loadEncodeFunc(v.Elem().Type())(v.Elem())
		}
	case reflect.String:
		return func(v reflect.Value) ([]byte, error) {
			var bytes []byte

			bytes = append(bytes, EncodeUint8(binn.StringType)...)
			bytes = append(bytes, EncodeString(v.String())...)
			bytes = append(bytes, 0x00)

			return bytes, nil
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		return func(v reflect.Value) ([]byte, error) {
			var bytes []byte

			bytes = append(bytes, EncodeUint8(uint8(detectIntType(int(v.Int()))))...)
			bytes = append(bytes, EncodeInt(int(v.Int()))...)

			return bytes, nil
		}
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		return func(v reflect.Value) ([]byte, error) {
			var bytes []byte

			bytes = append(bytes, EncodeUint8(uint8(detectUintType(uint(v.Uint()))))...)
			bytes = append(bytes, EncodeUint(uint(v.Uint()))...)

			return bytes, nil
		}
	case reflect.Float32:
		return func(v reflect.Value) ([]byte, error) {
			var bytes []byte

			bytes = append(bytes, EncodeUint8(binn.Float32Type)...)
			bytes = append(bytes, EncodeFloat32(float32(v.Float()))...)

			return bytes, nil
		}
	case reflect.Float64:
		return func(v reflect.Value) ([]byte, error) {
			var bytes []byte

			bytes = append(bytes, EncodeUint8(binn.Float64Type)...)
			bytes = append(bytes, EncodeFloat64(v.Float())...)

			return bytes, nil
		}
	case reflect.Slice, reflect.Array:
		return newArrayEncoder(t)
	case reflect.Ptr:
		return newPtrEncoder(t)
	}

	return func(v reflect.Value) ([]byte, error) {
		return nil, &UnsupportedTypeError{t}
	}
}
