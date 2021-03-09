// This source file is part of the EdgeDB open source project.
//
// Copyright 2020-present EdgeDB Inc. and the EdgeDB authors.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package codecs

import (
	"testing"
	"unsafe"

	"github.com/edgedb/edgedb-go/internal/buff"
	types "github.com/edgedb/edgedb-go/internal/edgedbtypes"
)

func BenchmarkDecodeUUID(b *testing.B) {
	data := []byte{
		0, 1, 2, 3, 3, 2, 1, 0, 8, 7, 6, 5, 5, 6, 7, 8,
	}
	r := buff.SimpleReader(data)

	var result types.UUID
	ptr := unsafe.Pointer(&result)
	codec := &UUID{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkEncodeUUID(b *testing.B) {
	w := buff.NewWriter([]byte{})
	id := types.UUID{1, 2, 3, 4, 5, 6, 7, 8, 8, 7, 6, 5, 4, 3, 2, 1}
	codec := &UUID{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = codec.Encode(w, id, Path(""))
	}
}

func BenchmarkDecodeString(b *testing.B) {
	data := []byte{104, 101, 108, 108, 111}
	r := buff.SimpleReader(data)

	var result string
	ptr := unsafe.Pointer(&result)
	codec := &Str{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkDecodeBytes(b *testing.B) {
	data := []byte{104, 101, 108, 108, 111}
	r := buff.SimpleReader(data)

	var result []byte
	ptr := unsafe.Pointer(&result)
	codec := &Bytes{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkDecodeInt16(b *testing.B) {
	data := []byte{1, 2}
	r := buff.SimpleReader(data)

	var result int16
	ptr := unsafe.Pointer(&result)
	codec := &Int16{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkDecodeInt32(b *testing.B) {
	data := []byte{1, 2, 3, 4}
	r := buff.SimpleReader(data)

	var result int32
	ptr := unsafe.Pointer(&result)
	codec := &Int32{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkDecodeInt64(b *testing.B) {
	data := []byte{1, 2, 3, 4, 5, 6, 7, 8}
	r := buff.SimpleReader(data)

	var result int64
	ptr := unsafe.Pointer(&result)
	codec := &Int64{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkDecodeFloat32(b *testing.B) {
	data := []byte{
		0xc2, 0, 0, 0,
	}
	r := buff.SimpleReader(data)

	var result float32
	ptr := unsafe.Pointer(&result)
	codec := &Float32{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkDecodeFloat64(b *testing.B) {
	data := []byte{
		0xc0, 0x50, 0, 0, 0, 0, 0, 0,
	}
	r := buff.SimpleReader(data)

	var result float64
	ptr := unsafe.Pointer(&result)
	codec := &Float64{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}

func BenchmarkDecodeBool(b *testing.B) {
	data := []byte{1}
	r := buff.SimpleReader(data)

	var result bool
	ptr := unsafe.Pointer(&result)
	codec := &Bool{}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		r.Buf = data
		codec.Decode(r, ptr)
	}
}
