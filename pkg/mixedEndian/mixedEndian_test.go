package mixedEndian

import (
	"bytes"
	"encoding/binary"
	"errors"
	"io"
	"testing"
)

type NoTagStruct struct {
	A uint8
	B int16
	C uint32
}

type TaggedStruct struct {
	A uint16 `endian:"big"`
	B uint16 `endian:"little"`
}

type NestedStruct struct {
	A uint16 `endian:"big"`
	B TaggedStruct
	C uint16 `endian:"little"`
}

func TestRead(t *testing.T) {
	type args struct {
		ioReader      io.Reader
		defaultEndian binary.ByteOrder
		data          any
	}

	reference := []byte{0x01, 0x23, 0x45, 0x67, 0x89, 0xAB, 0xCD, 0xEF}

	Notags := NoTagStruct{}
	tags := TaggedStruct{}
	nested := NestedStruct{}
	nonStruct := 0

	tests := []struct {
		name     string
		args     args
		wantData any
		wantErr  error
	}{
		{
			name: "no tags",
			args: args{
				ioReader:      bytes.NewReader(reference),
				defaultEndian: BigEndian,
				data:          Notags,
			},
			wantErr: nil,
			wantData: NoTagStruct{
				A: 0x01,
				B: 0x2345,
				C: 0x6789ABCD,
			},
		},
		{
			name: "tags",
			args: args{
				ioReader:      bytes.NewReader(reference),
				defaultEndian: BigEndian,
				data:          tags,
			},
			wantErr: nil,
			wantData: TaggedStruct{
				A: 0x0123,
				B: 0x6745,
			},
		},
		{
			name: "nested struct",
			args: args{
				ioReader:      bytes.NewReader(reference),
				defaultEndian: BigEndian,
				data:          nested,
			},
			wantErr: nil,
			wantData: NestedStruct{
				A: 0x0123,
				B: TaggedStruct{
					A: 0x4567,
					B: 0xAB89,
				},
				C: 0xEFCD,
			},
		},
		{
			name: "non-struct",
			args: args{
				ioReader:      bytes.NewReader(reference),
				defaultEndian: BigEndian,
				data:          nonStruct,
			},
			wantErr: ErrUnexpectedType,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if err := Read(tt.args.ioReader, tt.args.defaultEndian, &tt.args.data); tt.wantErr != nil && !errors.Is(err, tt.wantErr) {
				t.Errorf("Read() error = %v, wanted %v", err, tt.wantErr)
			} else if tt.wantData != nil {
				t.Errorf("Read() data = %v, wanted %v", tt.args.data, tt.wantData)
			}
		})
	}
}
