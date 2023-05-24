package pkg

import (
	"bytes"
	"encoding/binary"
	"fmt"
	"math"
)

func Int64ToBytes(num int64) []byte {
	b := make([]byte, 8)
	binary.BigEndian.PutUint64(b, uint64(num))
	return b
}

func BytesToInt64(bytes []byte) int64 {
	return int64(binary.BigEndian.Uint64(bytes))
}

func Float64ToBytes(f float64) []byte {
	bits := math.Float64bits(f)
	buffer := new(bytes.Buffer)
	_ = binary.Write(buffer, binary.LittleEndian, bits)
	return buffer.Bytes()
}

func BytesToFloat64(b []byte) float64 {
	buffer := bytes.NewReader(b)
	var bits uint64
	_ = binary.Read(buffer, binary.LittleEndian, &bits)
	return math.Float64frombits(bits)
}

func BoolToBytes(b bool) []byte {
	if b {
		return []byte{1}
	}
	return []byte{0}
}

func BytesToBool(b []byte) (bool, error) {
	if len(b) != 1 {
		return false, fmt.Errorf("input byte slice should have length 1")
	}
	if b[0] == 1 {
		return true, nil
	} else if b[0] == 0 {
		return false, nil
	}
	return false, fmt.Errorf("input byte slice should contain 0 or 1")
}
