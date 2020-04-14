package verthash

import (
	"bytes"
	"encoding/binary"
	"io/ioutil"

	"golang.org/x/crypto/sha3"
)

const VerthashHeaderSize uint32 = 80
const VerthashHashOutSize uint32 = 32
const VerthashP0Size uint32 = 64
const VerthashIter uint32 = 8
const VerthashSubset uint32 = VerthashP0Size * VerthashIter
const VerthashRotations uint32 = 32
const VerthashIndexes uint32 = 4096
const VerthashByteAlignment uint32 = 16

type Verthasher struct {
	datafile []byte
}

func NewVerthasher(dataFileLocation string) (*Verthasher, error) {
	b, err := ioutil.ReadFile(dataFileLocation)
	if err != nil {
		return nil, err
	}

	return &Verthasher{datafile: b}, nil
}

func fnv1a(a, b uint32) uint32 {
	return (a ^ b) * 0x1000193
}

func (vh *Verthasher) Hash(input []byte) [32]byte {
	p1 := [32]byte{}

	input_copy := make([]byte, len(input))
	copy(input_copy[:], input[:])

	sha3hash := sha3.Sum256(input_copy)

	copy(p1[:], sha3hash[:])
	p0 := make([]byte, VerthashSubset)
	for i := uint32(0); i < VerthashIter; i++ {
		input_copy[0] += 0x01
		digest64 := sha3.Sum512(input_copy)
		copy(p0[i*VerthashP0Size:], digest64[:])
	}

	buf := bytes.NewBuffer(p0)
	p0Index := make([]uint32, len(p0)/4)
	for i := 0; i < len(p0Index); i++ {
		binary.Read(buf, binary.LittleEndian, &p0Index[i])

	}

	seekIndexes := make([]uint32, VerthashIndexes)

	for x := uint32(0); x < VerthashRotations; x++ {
		copy(seekIndexes[x*VerthashSubset/4:], p0Index)
		for y := 0; y < len(p0Index); y++ {
			p0Index[y] = (p0Index[y] << 1) | (1 & (p0Index[y] >> 31))
		}
	}

	var valueAccumulator uint32
	var mdiv uint32
	mdiv = ((uint32(len(vh.datafile)) - VerthashHashOutSize) / VerthashByteAlignment) + 1
	valueAccumulator = uint32(0x811c9dc5)
	buf = bytes.NewBuffer(p1[:])
	p1Arr := make([]uint32, VerthashHashOutSize/4)
	for i := 0; i < len(p1Arr); i++ {
		binary.Read(buf, binary.LittleEndian, &p1Arr[i])
	}
	for i := uint32(0); i < VerthashIndexes; i++ {
		offset := (fnv1a(seekIndexes[i], valueAccumulator) % mdiv) * VerthashByteAlignment
		for i2 := uint32(0); i2 < VerthashHashOutSize/4; i2++ {
			value := binary.LittleEndian.Uint32(vh.datafile[offset+i2*4 : offset+((i2+1)*4)])
			p1Arr[i2] = fnv1a(p1Arr[i2], value)
			valueAccumulator = fnv1a(valueAccumulator, value)
		}
	}

	for i := uint32(0); i < VerthashHashOutSize/4; i++ {
		binary.LittleEndian.PutUint32(p1[i*4:], p1Arr[i])
	}

	return p1
}
