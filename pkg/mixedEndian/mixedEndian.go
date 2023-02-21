// mixedEndian is inspired by the encoding/binary package's Read() and Write() functions
// with the ability to specify endianness at a field-level through struct tagging.
//
// struct tags should be used with a key of "endian" and values of either "little" or "big" for example:
//
//	type abc struct {
//		a uint16
//		b uint16 `endian:"little"`
//		c uint32 `endian:"big"`
//	}
package mixedEndian

import (
	"encoding/binary"
	"fmt"
	"io"
	"reflect"
)

var (
	// Alias for binary.BigEndian to reduce the need for multiple imports
	BigEndian = binary.BigEndian

	// Alias for binary.LittleEndian to reduce the need for multiple imports
	LittleEndian = binary.LittleEndian

	// Error wrapped to specify unexpected types encountered during reflection
	ErrUnexpectedType = fmt.Errorf("Unexpected type.")
)

type reader struct {
	r io.Reader
	o binary.ByteOrder
}

func Read(ioReader io.Reader, defaultEndian binary.ByteOrder, data *any) (err error) {

	r := reader{
		r: ioReader,
		o: defaultEndian,
	}

	return r.readOrdered(reflect.ValueOf(*data), defaultEndian)
}

func (r *reader) readOrdered(v reflect.Value, o binary.ByteOrder) (err error) {
	switch k := v.Kind(); k {
	// Structs
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			// Slightly slower, but very much needed
			if f := v.Field(i); f.CanSet() && t.Field(i).Name != "_" {
				// Get endian tag if set
				targetEndian := o
				switch t.Field(i).Tag.Get("endian") {
				case "big":
					targetEndian = BigEndian
				case "little":
					targetEndian = LittleEndian
				}

				if err = r.readOrdered(f, targetEndian); err != nil {
					return
				}
			}
		}

	// List types
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err = r.readOrdered(v.Index(i), r.o); err != nil {
				return
			}
		}

	// Base types
	case reflect.Bool,
		reflect.Int,
		reflect.Uint,
		reflect.Int8,
		reflect.Uint8,
		reflect.Int16,
		reflect.Uint16,
		reflect.Int32,
		reflect.Uint32,
		reflect.Int64,
		reflect.Uint64:
		bs := make([]byte, size(k))
		if _, err = io.ReadFull(r.r, bs); err != nil {
			return
		}

		switch k {
		case reflect.Bool:
			v.SetBool(bs[0] != 0)
		case reflect.Uint8:
			v.SetUint(uint64(bs[0]))
		case reflect.Int8:
			v.SetInt(int64(int8(bs[0])))
		case reflect.Uint16:
			v.SetUint(uint64(o.Uint16(bs)))
		case reflect.Int16:
			v.SetInt(int64(int16(o.Uint16(bs))))
		case reflect.Uint32:
			v.SetUint(uint64(o.Uint32(bs)))
		case reflect.Int32:
			v.SetInt(int64(int32(o.Uint32(bs))))
		case reflect.Uint64:
			v.SetUint(o.Uint64(bs))
		case reflect.Int64:
			v.SetInt(int64(o.Uint64(bs)))
		}

	// Unknown type
	default:
		return fmt.Errorf("%w Expected int, uint, bool, array, slice, or struct; Got %s", ErrUnexpectedType, v.Type().String())
	}

	return
}

type writer struct {
	w io.Writer
	o binary.ByteOrder
}

func Write(ioWriter io.Writer, defaultEndian binary.ByteOrder, data any) (err error) {
	w := writer{
		w: ioWriter,
		o: defaultEndian,
	}

	return w.writeOrdered(reflect.ValueOf(data), defaultEndian)
}

func (w *writer) writeOrdered(v reflect.Value, o binary.ByteOrder) (err error) {
	switch k := v.Kind(); k {
	// Structs
	case reflect.Struct:
		t := v.Type()
		for i := 0; i < v.NumField(); i++ {
			// Get endian tag if set, else default
			targetEndian := o
			switch t.Field(i).Tag.Get("endian") {
			case "little":
				targetEndian = LittleEndian
			case "big":
				targetEndian = BigEndian
			}

			if err = w.writeOrdered(v.Field(i), targetEndian); err != nil {
				return
			}
		}

	// List types
	case reflect.Slice, reflect.Array:
		for i := 0; i < v.Len(); i++ {
			if err = w.writeOrdered(v.Index(i), w.o); err != nil {
				return
			}
		}

	// Base types
	case reflect.Bool,
		reflect.Int,
		reflect.Uint,
		reflect.Int8,
		reflect.Uint8,
		reflect.Int16,
		reflect.Uint16,
		reflect.Int32,
		reflect.Uint32,
		reflect.Int64,
		reflect.Uint64:
		bs := make([]byte, size(k))

		switch k {
		case reflect.Bool:
			if v.Bool() {
				bs[0] = 1
			} else {
				bs[0] = 0
			}
		case reflect.Uint8:
			bs[0] = uint8(v.Uint())
		case reflect.Int8:
			bs[0] = uint8(int8(v.Int()))
		case reflect.Uint16:
			o.PutUint16(bs, uint16(v.Uint()))
		case reflect.Int16:
			o.PutUint16(bs, uint16(int16(v.Int())))
		case reflect.Uint32:
			o.PutUint32(bs, uint32(v.Uint()))
		case reflect.Int32:
			o.PutUint32(bs, uint32(int32(v.Int())))
		case reflect.Uint64:
			o.PutUint64(bs, v.Uint())
		case reflect.Int64:
			o.PutUint64(bs, uint64(v.Int()))
		}

		if _, err = w.w.Write(bs); err != nil {
			return
		}

	// Unknown type
	default:
		return fmt.Errorf("%w Expected int, uint, bool, array, slice, or struct; Got %s", ErrUnexpectedType, v.Type().String())
	}

	return
}

// size is a dumb function, and should already exist as a part of reflect/value
func size(k reflect.Kind) int {
	switch k {
	case reflect.Bool,
		reflect.Int8,
		reflect.Uint8:
		return 1
	case reflect.Int16,
		reflect.Uint16:
		return 2
	case reflect.Int32,
		reflect.Uint32:
		return 4
	case reflect.Int64,
		reflect.Uint64:
		return 8
	default:
		return 0
	}
}
