package simplerpc

import (
	"fmt"
	"math/rand"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSerializeInteger(t *testing.T) {
	test1 := func(v int64) []byte {
		return SerializeInteger([]byte{}, v)
	}

	// mode 00:
	assert.Equal(t, []byte{0x00}, test1(0x0))   // 0 00: 0 0x0
	assert.Equal(t, []byte{0x01}, test1(0x1))   // 0 00: 0 0x1
	assert.Equal(t, []byte{0x1e}, test1(0x1e))  // 0 00: 1 0xe
	assert.Equal(t, []byte{0x1f}, test1(0x1f))  // 0 00: 1 0xf
	assert.Equal(t, []byte{0x80}, test1(-0x1))  // 1 00: 0 0x0
	assert.Equal(t, []byte{0x9f}, test1(-0x20)) // 1 00: 1 0xf

	// mode 01:
	assert.Equal(t, []byte{0x20, 0x20}, test1(0x20))    // 0 01: 0 0x0 20
	assert.Equal(t, []byte{0x20, 0x64}, test1(0x64))    // 0 01: 0 0x0 64
	assert.Equal(t, []byte{0x23, 0xe8}, test1(0x3e8))   // 0 01: 0 0x3 e8
	assert.Equal(t, []byte{0x3f, 0x40}, test1(0x1f40))  // 0 01: 1 0xf 40
	assert.Equal(t, []byte{0x3f, 0xff}, test1(0x1fff))  // 0 01: 1 0xf ff
	assert.Equal(t, []byte{0xa4, 0xd1}, test1(-0x4d2))  // 1 01: 0 0x4 d1
	assert.Equal(t, []byte{0xbf, 0xff}, test1(-0x2000)) // 1 01: 1 0xf ff

	// mode 10:
	assert.Equal(t, []byte{0x40, 0x20, 0x00}, test1(0x2000))    // 0 10: 0 0x0 20 00
	assert.Equal(t, []byte{0x40, 0x99, 0x99}, test1(0x9999))    // 0 10: 0 0x0 99 99
	assert.Equal(t, []byte{0x5f, 0xff, 0xff}, test1(0x1fffff))  // 0 10: 1 0xf ff ff
	assert.Equal(t, []byte{0xdf, 0xff, 0xff}, test1(-0x200000)) // 1 10: 1 0xf ff ff
	assert.Equal(t, []byte{0xc8, 0x76, 0x53}, test1(-0x87654))  // 1 10: 0 0x8 76 53

	// mode 11:
	assert.Equal(t, []byte{0x68, 0x20, 0x00, 0x00}, test1(0x200000))                                          // 0 11 010: 00 0x20 00 00
	assert.Equal(t, []byte{0x68, 0x23, 0x45, 0x67}, test1(0x234567))                                          // 0 11 010: 00 0x23 45 67
	assert.Equal(t, []byte{0x69, 0x23, 0x45, 0x67}, test1(0x1234567))                                         // 0 11 010: 01 0x23 45 67
	assert.Equal(t, []byte{0x6b, 0x23, 0x45, 0x67}, test1(0x3234567))                                         // 0 11 010: 11 0x23 45 67
	assert.Equal(t, []byte{0x6c, 0x32, 0x34, 0x56, 0x78}, test1(0x32345678))                                  // 0 11 011: 00 0x32 34 56 78
	assert.Equal(t, []byte{0x7c, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xef}, test1(0x1234567890abcdef))  // 0 11 111: 00 0x12 34 56 78 90 ab cd ef
	assert.Equal(t, []byte{0xe9, 0x23, 0x45, 0x66}, test1(-0x1234567))                                        // 1 11 010: 01 0x23 45 66
	assert.Equal(t, []byte{0xfc, 0x12, 0x34, 0x56, 0x78, 0x90, 0xab, 0xcd, 0xee}, test1(-0x1234567890abcdef)) // 1 11 111: 00 0x12 34 56 78 90 ab cd ee
}

func TestDeserializeInteger(t *testing.T) {
	test1 := func(bytes ...byte) int64 {
		newbuf, ret := DeserializeInteger(append(bytes, 0))
		l := len(newbuf)
		if l != 1 {
			panic(fmt.Sprintf("failed to deserialize integer, l=%d", l))
		}
		return ret
	}

	// mode 00
	assert.EqualValues(t, 0x00, test1(0x00))  // 0 00: 0 0x0
	assert.EqualValues(t, 0x0a, test1(0x0a))  // 0 00: 0 0xa
	assert.EqualValues(t, 0x1f, test1(0x1f))  // 0 00: 1 0xf
	assert.EqualValues(t, -0x20, test1(0x9f)) // 1 00: 1 0xf
	assert.EqualValues(t, -0x2, test1(0x81))  // 1 00: 0 0x1

	// mode 01
	assert.EqualValues(t, 0xed, test1(0x20, 0xed))    // 0 01: 0 0x0 ed
	assert.EqualValues(t, 0xedc, test1(0x2e, 0xdc))   // 0 01: 0 0xe dc
	assert.EqualValues(t, 0x1edc, test1(0x3e, 0xdc))  // 0 01: 1 0xe dc
	assert.EqualValues(t, -0x1edc, test1(0xbe, 0xdb)) // 1 01: 1 0xe db
	assert.EqualValues(t, 0x5, test1(0x20, 0x05))     // 0 01: 0 0x0 05 - can be represented as mode 00, but parsing should work for this too

	// mode 10
	assert.EqualValues(t, 0x2000, test1(0x40, 0x20, 0x00))    // 0 10: 0 0x0 20 00
	assert.EqualValues(t, 0x23456, test1(0x42, 0x34, 0x56))   // 0 10: 0 0x2 34 56
	assert.EqualValues(t, 0x123456, test1(0x52, 0x34, 0x56))  // 0 10: 1 0x2 34 56
	assert.EqualValues(t, -0x123456, test1(0xd2, 0x34, 0x55)) // 1 10: 1 0x2 34 55
	assert.EqualValues(t, -0x200000, test1(0xdf, 0xff, 0xff)) // 1 10: 1 0xf ff ff
	assert.EqualValues(t, 0x5, test1(0x40, 0x00, 0x05))       // 0 01: 0 0x0 00 05 - can be represented as mode 00 or 01, but parsing should work for this too
	assert.EqualValues(t, 0xff, test1(0x40, 0x00, 0xff))      // 0 01: 0 0x0 00 ff - can be represented as mode 01, but parsing should work for this too

	// mode 11
	assert.EqualValues(t, 0x234567, test1(0x68, 0x23, 0x45, 0x67))                                          // 0 11 010: 00 0x23 45 67
	assert.EqualValues(t, 0x3234567, test1(0x6b, 0x23, 0x45, 0x67))                                         // 0 11 010: 11 0x23 45 67
	assert.EqualValues(t, 0x1234567890abcde, test1(0x79, 0x23, 0x45, 0x67, 0x89, 0x0a, 0xbc, 0xde))         // 0 11 110: 01 0x23 45 67 89 0a bc de
	assert.EqualValues(t, 0x7890abcdef123456, test1(0x7c, 0x78, 0x90, 0xab, 0xcd, 0xef, 0x12, 0x34, 0x56))  // 0 11 111: 00 0x78 90 ab cd ef 12 34 56
	assert.EqualValues(t, -0x8000000000000000, test1(0xfc, 0x7f, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff, 0xff)) // 1 11 111: 00 0x7f ff ff ff ff ff ff ff

	// mode 11 - these can be represented on lower modes too
	assert.EqualValues(t, 0x11, test1(0x60, 0x11))                         // 0 11 000: 00 0x11
	assert.EqualValues(t, 0x11, test1(0x64, 0x00, 0x11))                   // 0 11 001: 00 0x00 11
	assert.EqualValues(t, 0x11, test1(0x70, 0x00, 0x00, 0x00, 0x00, 0x11)) // 0 11 100: 00 0x00 00 00 00 11

	// invalid empty buffer
	newbuf, _ := DeserializeInteger([]byte{})
	assert.Nil(t, newbuf)

	// invalid mode 01: at least 2 bytes are needed
	newbuf, _ = DeserializeInteger([]byte{0x20 /*0 01 00000*/})
	assert.Nil(t, newbuf)
	newbuf, _ = DeserializeInteger([]byte{0x20 /*0 01 00000*/, 0x00})
	assert.NotNil(t, newbuf) // ok

	// invalid mode 10: at least 3 bytes are needed
	newbuf, _ = DeserializeInteger([]byte{0x40 /*0 10 00000*/})
	assert.Nil(t, newbuf)
	newbuf, _ = DeserializeInteger([]byte{0x40 /*0 10 00000*/, 0x00})
	assert.Nil(t, newbuf)
	newbuf, _ = DeserializeInteger([]byte{0x40 /*0 10 00000*/, 0x00, 0x00})
	assert.NotNil(t, newbuf) // ok

	// invalid mode 11, size=1: at least 2 bytes are needed
	newbuf, _ = DeserializeInteger([]byte{0x60 /*0 11 000 00*/})
	assert.Nil(t, newbuf)
	newbuf, _ = DeserializeInteger([]byte{0x60 /*0 11 000 00*/, 0x00})
	assert.NotNil(t, newbuf) // ok

	// invalid mode 11, size=6: at least 7 bytes are needed
	newbuf, _ = DeserializeInteger([]byte{0x74 /*0 11 101 00*/, 0x00, 0x00, 0x00, 0x00, 0x00})
	assert.Nil(t, newbuf)
	newbuf, _ = DeserializeInteger([]byte{0x74 /*0 11 101 00*/, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	assert.NotNil(t, newbuf) // ok

	// invalid mode 11, size=8 with invalid '1' bit at start
	newbuf, _ = DeserializeInteger([]byte{0x7c /*0 11 111 00*/, 0x80, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	assert.Nil(t, newbuf)
	newbuf, _ = DeserializeInteger([]byte{0x7d /*0 11 111 01*/, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00, 0x00})
	assert.Nil(t, newbuf) // ok
}

func TestIntegerSerializationWithRandomNumbers(t *testing.T) {
	var arr [12]byte
	for i := 0; i < 1000000; i++ {
		n := rand.Int63()
		if rand.Int()%2 == 0 {
			n = -n
		}
		buf := SerializeInteger(arr[:0], n)
		nb, newn := DeserializeInteger(buf)
		assert.NotNil(t, nb, "for ", n)
		assert.Equal(t, n, newn)
	}
}

func TestSerializeString(t *testing.T) {
	test1 := func(str string) []byte {
		return SerializeString([]byte{}, str)
	}

	assert.Equal(t, []byte{0x00}, test1(""))
	assert.Equal(t, []byte{0x02, 'h', 'i'}, test1("hi"))
	assert.Equal(t, []byte{0x10, 'q', 'w', 'e', 'r', 't', 'y', 'u', 'i', 'o', 'p', 'a', 's', 'd', 'f', 'g', 'h'}, test1("qwertyuiopasdfgh"))
	assert.Nil(t, SerializeString(nil, "does nothing with nil input"))
}

func TestDeserializeString(t *testing.T) {
	test1 := func(bytes ...byte) string {
		newbuf, ret := DeserializeString(append(bytes, 0))
		l := len(newbuf)
		if l != 1 {
			panic(fmt.Sprintf("failed to deserialize string, l=%d", l))
		}
		return ret
	}

	assert.Equal(t, "", test1(0x00))
	assert.Equal(t, "Hi!", test1(0x03, 'H', 'i', '!'))
	assert.Equal(t, "asdfg", test1(0x05, 'a', 's', 'd', 'f', 'g'))

	// invalid: nil input
	newbuf, _ := DeserializeString(nil)
	assert.Nil(t, newbuf)

	// invalid: length parse failed (mode 11, next byte would be needed)
	newbuf, _ = DeserializeString([]byte{0x60})
	assert.Nil(t, newbuf)

	// invalid: not that many bytes as the given length
	newbuf, _ = DeserializeString([]byte{0x3, 'a', 'b'})
	assert.Nil(t, newbuf)
}
