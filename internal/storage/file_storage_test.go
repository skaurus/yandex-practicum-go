package storage

import (
	"bytes"
	"fmt"
	"strings"
	"testing"
)

func BenchmarkMarshalJSONBuilders(b *testing.B) {
	benchmarks := []struct {
		name   string
		testFn func(int) []byte
		param  int
	}{
		{"strings.Builder for []bytes concat", stringsBuilderBytesConcat, 10},
		{"bytes.Buffer for []bytes concat", bytesBufferBytesConcat, 10},
		{"strings.Builder for []bytes concat", stringsBuilderBytesConcat, 50},
		{"bytes.Buffer for []bytes concat", bytesBufferBytesConcat, 50},
		{"strings.Builder for []bytes concat", stringsBuilderBytesConcat, 500},
		{"bytes.Buffer for []bytes concat", bytesBufferBytesConcat, 500},
		{"strings.Builder for []bytes concat", stringsBuilderBytesConcat, 5000},
		{"bytes.Buffer for []bytes concat", bytesBufferBytesConcat, 5000},
	}
	for _, bm := range benchmarks {
		b.Run(fmt.Sprintf("%s, %d rows", bm.name, bm.param), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				bm.testFn(bm.param)
			}
		})
	}
}

var byteString = []byte(`[11,"https://ya.ru/search","skaurus"]`)

func stringsBuilderBytesConcat(param int) []byte {
	var sb strings.Builder
	for ; param > 0; param-- {
		sb.Write(byteString)
		sb.WriteString(",")
	}
	return []byte(sb.String())
}

func bytesBufferBytesConcat(param int) []byte {
	var bytesBuffer bytes.Buffer
	for ; param > 0; param-- {
		bytesBuffer.Write(byteString)
		bytesBuffer.WriteString(",")
	}
	return bytesBuffer.Bytes()
}
