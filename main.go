package main

import (
	"bufio"
	"bytes"
	"compress/gzip"
	"encoding/binary"
	"fmt"
	"io"
	"os"

	"github.com/go-audio/audio"
	"github.com/go-audio/wav"
)

var (
	mapFrom = make(map[int]int, 1024)
	mapTo   = make(map[int]int, 1024)

	//doZip = true // easily toggle-able because zip makes it worse sometimes
)

//func main() {
//	readFromSet()
//
//	encode := false
//
//	operate(os.Args[1], os.Args[2], encode)
//}

func main() {
	readFromSet()

	in1 := readWAV("1.wav")
	out := encode2(in1)
	write(out, "out.bin")

	in2 := read("out.bin")
	decoded := decode2(in2)
	writeAsWav(decoded, "out.wav")
}

//func main() {
//
//	readFromSet()
//
//	total := 0.0
//
//	// read every .wav inside data/ and measure its entropy
//	files, err := os.ReadDir("data")
//	if err != nil {
//		panic(err)
//	}
//	for _, file := range files {
//		if file.IsDir() {
//			continue
//		}
//		name := file.Name()
//		if name[len(name)-4:] == ".wav" {
//			sz := getTheoreticalCompressedSize("data/" + name)
//			total += sz
//			fmt.Println(name, sz)
//		}
//	}
//
//	fmt.Println("total", total)
//}
//
//func getTheoreticalCompressedSize(file string) float64 {
//	data := readWAV(file)
//	data = mapTo10Bit(data)
//	diff := differentiate(data)
//	e := getEntropy(diff)
//	return float64(len(diff)) * e / 8.0
//}

func operate(infile string, outfile string, encode bool) {
	if encode {
		in := readWAV(infile)
		encoded := encode2(in)
		write(encoded, outfile)
	} else {
		in := read(infile)
		decoded := decode2(in)
		writeAsWav(decoded, outfile)
	}
}

func read(filename string) []byte {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	buf := new(bytes.Buffer)
	_, err = io.Copy(buf, f)
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}

func write(out []byte, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	n, err := f.Write(out)
	if err != nil {
		panic(err)
	}
	if n != len(out) {
		panic("not enough bytes written")
	}
}

func readWAV(filename string) []int {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)
	//decoder.ReadInfo()
	//fmt.Println(decoder)

	intBuf, err := decoder.FullPCMBuffer()
	if err != nil {
		panic(err)
	}

	return intBuf.Data
}

func encode2(data []int) []byte {
	// diff, shift to positive, then record start, then huffman encode the rest of the differences
	data = mapTo10Bit(data)
	diff := differentiate(data)
	//diff = differentiate(diff[1:])
	//diff[0] = 0 // first sample is the same
	shift := -minimum(diff)
	diff = vecaddscalar(diff, shift) // make non-negative
	start := diff[0]
	diff = diff[1:] // remove start

	fmt.Println("shift", shift)
	fmt.Println("diff len", len(diff))
	//fmt.Println(start)
	//fmt.Println(chunkLen, maxabs)
	//fmt.Println(diff)

	//arithEncode(diff)

	//fmt.Println(bwt(diff))

	saveBytes(huffman(mtf(bwt(data))))
	//fmt.Println(len(huffman(mtf(bwt(data)))))

	huff := huffman(diff)

	//ae := acesEncode(diff, byte(chunkLen))
	fmt.Println("full huff length", len(huff))

	result := make([]byte, 0, len(huff)+2*binary.MaxVarintLen64)
	result = addVarints(result, uint64(shift), uint64(start))
	result = append(result, huff...)

	//fmt.Println(result)

	fmt.Println("encoded pre-zip", len(result))
	new := zip(result)
	fmt.Println("encoded after-zip", len(result))
	if len(new) < len(result) {
		new = append([]byte{1}, new...) // 1 means zipped
		return new
	} else {
		result = append([]byte{0}, result...) // 0 means not zipped
		return result
	}
}

func decode2(data []byte) []int {
	if data[0] == 1 {
		data = data[1:]
		data = unzip(data)
	} else {
		data = data[1:]
	}

	shift, start, data := get2Varints(data)
	diff := unHuffman(data)
	diff = append([]int{int(start)}, diff...)
	diff = vecaddscalar(diff, -int(shift))
	diff = integrate(diff)
	diff = mapTo16Bit(diff)
	return diff
}

//func acesEncode(in []int, chunkLen byte) []byte {
//	buf := new(bytes.Buffer)
//	bw := aces.NewBitWriter(chunkLen, buf)
//	for _, num := range in {
//		bw.Write(byte(num))
//	}
//	bw.Flush()
//	return buf.Bytes()
//}
//
//func acesDecode(in []byte, chunkLen byte) []int {
//	buf := bytes.NewReader(in)
//	br, err := aces.NewBitReader(chunkLen, buf)
//	if err != nil {
//		panic(err)
//	}
//	out := make([]int, 0, len(in)) // the length can be calculated based on chunklen later.
//	for {
//		b, err := br.Read()
//		if err != nil {
//			if err == io.EOF {
//				break
//			}
//			panic(err)
//		}
//		out = append(out, int(b))
//	}
//	return out
//}

func addVarints(buf []byte, a ...uint64) []byte {
	for _, num := range a {
		buf = binary.AppendUvarint(buf, num)
	}
	return buf
}

func get2Varints(buf []byte) (one, two uint64, newbuf []byte) {
	out := make([]uint64, 2)
	for i := 0; i < 2; i++ {
		num, n := binary.Uvarint(buf)
		out[i] = num
		buf = buf[n:]
	}
	return out[0], out[1], buf
}

func minimum(a []int) int {
	min := a[0]
	for _, num := range a {
		if num < min {
			min = num
		}
	}
	return min
}

func maximumAbs(a []int) int {
	m := max(a[0], -a[0])
	for _, num := range a {
		num = max(num, -num)
		if num > m {
			m = num
		}
	}
	return m
}

func vecaddscalar(a []int, b int) []int {
	for i := 0; i < len(a); i++ {
		a[i] += b
	}
	return a
}

func encode(filename string) []byte {
	f, err := os.Open(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	decoder := wav.NewDecoder(f)
	//decoder.ReadMetadata()
	decoder.ReadInfo()
	fmt.Println(decoder.NumChans)
	fmt.Println(decoder.SampleRate)
	fmt.Println(decoder.BitDepth)

	intBuf, err := decoder.FullPCMBuffer()
	if err != nil {
		panic(err)
	}

	data := intBuf.Data

	//fmt.Println(data)
	data = mapTo10Bit(data)

	//data = differentiate(data) // store diffs of audio samples instead of the samples themselves

	fmt.Println(data)

	numSamples := len(data)

	//fmt.Println(data)
	eight, two := tenBitTo8Bit2Bit(data)
	fmt.Println(two)
	fmt.Println(eight)
	samplesEncoded := make([]byte, binary.MaxVarintLen64)
	// so that when reading we know when 8 bit portion ends and 2 bit portion starts
	n := binary.PutVarint(samplesEncoded, int64(numSamples))
	samplesEncoded = samplesEncoded[:n]

	buf := new(bytes.Buffer)
	buf.Grow(len(eight) + len(two)) // more than enough.

	buf.Write(samplesEncoded)

	//buf.Grow(len(eight) + len(two))
	for _, b := range eight {
		buf.WriteByte(b)
	}

	// this is the part we can optimize super well
	tbw := NewTwoBitWriter(len(two))
	for _, b := range two {
		tbw.Write(b)
	}
	buf.Write(tbw.GetBytes())

	return zip(buf.Bytes())
}

func tenBitTo8Bit2Bit(data []int) (eight []byte, two []byte) {
	eight = make([]byte, len(data))
	two = make([]byte, len(data))
	for i := 0; i < len(data); i++ {
		eight[i] = byte(data[i] >> 2)
		two[i] = byte(data[i] & 0b11)
	}
	return
}

func decode(data []byte) []int {
	data = unzip(data)
	bufReader := bufio.NewReaderSize(bytes.NewReader(data), 1024*10)
	numSamples, err := binary.ReadVarint(bufReader)
	if err != nil {
		panic(err)
	}

	eight := make([]byte, numSamples)
	n, err := io.ReadFull(bufReader, eight)
	if n != int(numSamples) {
		panic("not enough bytes read")
	}
	if err != nil {
		panic(err)
	}

	two := make([]byte, numSamples)

	rest := make([]byte, 0, len(data))
	for { // optimize later lol
		b, err := bufReader.ReadByte()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
		rest = append(rest, b)
	}

	tbr := NewTwoBitReader(rest)
	for i := 0; i < len(two); i++ {
		two[i], err = tbr.Read()
		if err != nil {
			if err == io.EOF {
				break
			}
			panic(err)
		}
	}

	fmt.Println("two len", len(two))
	fmt.Println("eight len", len(eight))
	data2 := eightBit2BitTo10Bit(eight, two)
	//fmt.Println(data2)
	//data2 = integrate(data2) // recover the original samples

	data2 = mapTo16Bit(data2)
	writeAsWav(data2, "out.wav")
	return data2
}

func writeAsWav(data []int, filename string) {
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	encoder := wav.NewEncoder(f, 19531, 16, 1, 1)

	buf := audio.IntBuffer{
		Format: &audio.Format{
			NumChannels: 1,
			SampleRate:  19531,
		},
		Data:           data,
		SourceBitDepth: 16,
	}

	err = encoder.Write(&buf)
	if err != nil {
		panic(err)
	}
	err = encoder.Close()
	if err != nil {
		panic(err)
	}
}

func eightBit2BitTo10Bit(eight []byte, two []byte) []int {
	data := make([]int, len(eight))
	for i := 0; i < len(eight); i++ {
		data[i] = int(eight[i])<<2 + int(two[i])
	}
	return data
}

func mapTo10Bit(data []int) []int {
	var ok bool
	for i := 0; i < len(data); i++ {
		data[i], ok = mapTo[data[i]]
		if !ok {
			panic("not found")
		}
	}
	//data = differentiate(data)
	return data
}

func mapTo16Bit(data []int) []int {
	//data = integrate(data)
	//fmt.Println("before", data)
	for i := 0; i < len(data); i++ {
		data[i] = mapFrom[data[i]]
	}
	//fmt.Println("after", data)
	return data
}

func differentiate(data []int) []int {
	out := make([]int, len(data))
	out[0] = data[0] // first sample is the same
	for i := 1; i < len(data); i++ {
		out[i] = data[i] - data[i-1] // sample at i is the increase over the previous sample
	}
	return out
}

func integrate(data []int) []int {
	out := make([]int, len(data))
	out[0] = data[0] // first sample is the same
	for i := 1; i < len(data); i++ {
		out[i] = data[i] + out[i-1] // recover the original data
	}
	return out
}

func TestDifferentiateIntegrate() {
	randslice := []int{1, 2, 3, 4, 5, 6, 7, 8, 9, 10}
	fmt.Println(randslice)
	randslice = differentiate(randslice)
	fmt.Println(randslice)
	randslice = integrate(randslice)
	fmt.Println(randslice)
}

func readFromSet() {
	mapFrom = make(map[int]int)
	fileName := "set.txt"
	f, err := os.Open(fileName)
	if err != nil {
		panic(err)
	}
	defer f.Close()

	//i := -512
	i := 0
	for {
		var out int
		_, err := fmt.Fscanf(f, "%d\n", &out)
		if err != nil {
			break
		}
		mapFrom[i] = out
		mapTo[out] = i
		//fmt.Println(i, out)
		i++
	}
	fmt.Println("done reading from set.txt")
}

type TwoBitReader struct {
	buf    []byte
	pos    int
	bitIdx byte
}

func NewTwoBitReader(buf []byte) *TwoBitReader {
	return &TwoBitReader{buf: buf, pos: 0, bitIdx: 0}
}

func (r *TwoBitReader) Read() (byte, error) {
	if r.pos >= len(r.buf) {
		return 0, io.EOF
	}
	b := sliceByteLen(r.buf[r.pos], r.bitIdx, 2)
	r.bitIdx += 2
	if r.bitIdx == 8 {
		r.bitIdx = 0
		r.pos++
	}
	return b, nil
}

// sliceByteLen slices the byte b such that the result has length len and starting bit start
func sliceByteLen(b byte, start uint8, len uint8) byte {
	return (b << start) >> (8 - len)
}

type TwoBitWriter struct {
	buf    []byte
	pos    int
	bitIdx byte
}

func NewTwoBitWriter(length int) *TwoBitWriter {
	return &TwoBitWriter{buf: make([]byte, length)}
}

func (w *TwoBitWriter) Write(b byte) {
	spaceOnRight := 8 - w.bitIdx
	b = b << (spaceOnRight - 2) // whift byte
	w.buf[w.pos] |= b
	w.bitIdx += 2
	if w.bitIdx == 8 {
		w.bitIdx = 0
		w.pos++
	}
}

func (w *TwoBitWriter) GetBytes() []byte {
	return w.buf[:w.pos+1]
}

func unzip(data []byte) []byte {
	buf := bytes.NewBuffer(data)
	z, err := gzip.NewReader(buf)
	if err != nil {
		panic(err)
	}
	defer z.Close()

	out := new(bytes.Buffer)
	_, err = io.Copy(out, z)
	if err != nil {
		panic(err)
	}
	return out.Bytes()
}

func zip(data []byte) []byte {
	buf := new(bytes.Buffer)
	z, err := gzip.NewWriterLevel(buf, gzip.BestCompression)
	if err != nil {
		panic(err)
	}
	_, err = z.Write(data)
	if err != nil {
		panic(err)
	}
	err = z.Close()
	if err != nil {
		panic(err)
	}
	return buf.Bytes()
}
