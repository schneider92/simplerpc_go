package simplerpc

const (
	integer_sermode_00   = 0x00
	integer_sermode_01   = 0x20
	integer_sermode_10   = 0x40
	integer_sermode_11   = 0x60
	integer_sermode_mask = integer_sermode_11
)

// Serialize an integer to the end of buf and return the new buffer
func SerializeInteger(buf []byte, v int64) []byte {
	// check sign
	signmask := byte(0)
	var uv uint64
	if v < 0 {
		uv = uint64(-(v + 1))
		signmask = 0x80
	} else {
		uv = uint64(v)
	}

	// check if fits in mode 00
	if uv < 0x20 {
		b := signmask | byte(uv) | integer_sermode_00
		return append(buf, b)
	}

	// check if fits in mode 01
	if uv < 0x2000 {
		b1 := byte(uv & 0xff)
		b0 := byte(uv>>8) | signmask | integer_sermode_01
		return append(buf, b0, b1)
	}

	// check if fits in mode 10
	if uv < 0x200000 {
		b2 := byte(uv & 0xff)
		uv >>= 8
		b1 := byte(uv & 0xff)
		b0 := byte(uv>>8) | signmask | integer_sermode_10
		return append(buf, b0, b1, b2)
	}

	// mode 11
	// write to a temporary buffer
	var tmpbuf [9]byte
	var firstbyte byte = signmask | integer_sermode_11
	bytecount := 0
	{
		for uv > 3 {
			bytecount++
			tmpbuf[9-bytecount] = byte(uv & 0xff)
			uv >>= 8
		}
		firstbyte |= byte(uv)
	}
	//if bytecount == 0 || bytecount > 8 {
	// this cannot happen
	//panic(fmt.Sprintf("mode 11 invalid bytecount, v=%d, bc=%d", v, bytecount))
	//}

	// add size to first byte
	firstbyte |= (byte(bytecount-1) << 2)

	// add first byte to tmpbuf
	st := 8 - bytecount
	tmpbuf[st] = firstbyte

	// append bytes
	return append(buf, tmpbuf[st:]...)
}

func deserializeIntegerImplNoSign(buf []byte) ([]byte, uint64) {
	// find mode
	b0 := buf[0]
	mode := b0 & integer_sermode_mask
	b0 &= 0x1f

	// mode 00
	if mode == integer_sermode_00 {
		return buf[1:], uint64(b0)
	}

	// mode 01
	if mode == integer_sermode_01 {
		if len(buf) < 2 {
			return nil, 0
		}
		ret := uint64(b0)*256 + uint64(buf[1])
		return buf[2:], ret
	}

	// mode 10
	if mode == integer_sermode_10 {
		if len(buf) < 3 {
			return nil, 0
		}
		ret := (uint64(b0)*256+uint64(buf[1]))*256 + uint64(buf[2])
		return buf[3:], ret
	}

	// mode 11
	// get size and check if we have enough bytes
	size := int((b0&0x1c)>>2) + 1
	buf = buf[1:]
	if len(buf) < size {
		return nil, 0
	}

	// start value
	v := uint64(b0 & 0x3)

	// if size is 8, the 2 bits in the first byte and the first bit
	// of the next byte must be 0 to fit into the 63-bit value
	if size == 8 && (v != 0 || buf[0]&0x80 != 0) {
		return nil, 0
	}

	// combine all bytes of the value
	for i := 0; i < size; i++ {
		v = v*256 + uint64(buf[i])
	}

	// return result
	return buf[size:], v
}

// Deserialize an integer from the given buf and return the remaining bytes and
// the deserialized value. In case of an error (format error or nil input buffer),
// nil is returned
func DeserializeInteger(buf []byte) ([]byte, int64) {
	if len(buf) == 0 {
		return nil, 0
	}

	// get negative flag
	negative := buf[0]&0x80 == 0x80

	// deserialize as positive
	newbuf, uret := deserializeIntegerImplNoSign(buf)
	if newbuf == nil {
		return nil, 0
	}
	var ret int64
	if negative {
		ret = int64(uret)
		ret = -ret
		ret--
		//		ret = -int64(uret) - 1
	} else {
		ret = int64(uret)
	}
	return newbuf, ret
}

// Serialize a byte slice to the end of buf and return the new buffer
func SerializeBlob(buf []byte, data []byte) []byte {
	if buf == nil {
		return nil
	}
	buf = SerializeInteger(buf, int64(len(data)))
	buf = append(buf, data...)
	return buf
}

// Deserialize a byte slice from the given buf and return the remaining bytes and
// the deserialized value. In case of an error (format error or nil input buffer),
// nil is returned
func DeserializeBlob(buf []byte) (newbuf []byte, blobdata []byte) {
	var size int64
	buf, size = DeserializeInteger(buf)
	if buf == nil || size > int64(len(buf)) {
		return nil, nil
	}
	blobdata = buf[:size]
	newbuf = buf[size:]
	return
}

// Serialize a string to the end of buf and return the new buffer
func SerializeString(buf []byte, v string) []byte {
	return SerializeBlob(buf, []byte(v))
}

// Deserialize a string from the given buf and return the remaining bytes and
// the deserialized value. In case of an error (format error or nil input buffer),
// nil is returned
func DeserializeString(_buf []byte) (buf []byte, ret string) {
	buf = _buf
	var data []byte
	buf, data = DeserializeBlob(buf)
	if buf == nil {
		return
	}
	ret = string(data)
	return
}
