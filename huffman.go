package main

import (
	"bytes"
	"encoding/binary"
	"encoding/gob"
	"fmt"
	"math"
	"math/big"
	"sort"
	"strings"
)

type node struct {
	Symbol int
	count  int
	Left   *node
	Right  *node
}

type code struct {
	code uint64
	len  uint8
}

func (c code) String() string {
	//return fmt.Sprintf("%0*b %v %v", c.len, c.code, c.len, c.code)
	return fmt.Sprintf("%0*b", c.len, c.code)
}

func getProbabilities(data []int) map[int]int {
	counts := make(map[int]int)
	for _, sym := range data {
		counts[sym]++
	}
	return counts
}

func getHuffmanTree(probabilities map[int]int) *node {
	var nodes []*node
	for b, count := range probabilities {
		nodes = append(nodes, &node{Symbol: b, count: count})
	}
	sort.Slice(nodes, func(i, j int) bool {
		return nodes[i].count < nodes[j].count
	}) // sort ascending by count

	for len(nodes) > 1 {
		newInternal := &node{
			Left:   nodes[0],
			Right:  nodes[1],
			count:  nodes[0].count + nodes[1].count,
			Symbol: -1,
		}
		nodes = nodes[2:]
		if len(nodes) == 0 {
			nodes = append(nodes, newInternal)
			break
		}
		for i, n := range nodes {
			if n.count >= newInternal.count || i == len(nodes)-1 { // insert newInternal at the right place
				// add newInternal at index i+1
				nodes = append(nodes[:i+1], append([]*node{newInternal}, nodes[i+1:]...)...)
				break
			}
		}
	}
	//treePrint(nodes[0], 0)
	//dict := getDictionary(nodes[0])
	//dictPrint(dict)
	return nodes[0]
}

func getDictionary(n *node) map[int]code {
	dict := make(map[int]code)
	var traverse func(n *node, prefix code)
	traverse = func(n *node, prefix code) {
		if n.Symbol != -1 {
			dict[n.Symbol] = prefix
			return
		}
		leftcode := prefix
		leftcode.code <<= 1
		leftcode.len++
		traverse(n.Left, leftcode)
		rightcode := prefix
		rightcode.code <<= 1
		rightcode.code |= 1
		rightcode.len++
		traverse(n.Right, rightcode)
	}
	traverse(n, code{})
	return dict
}

func dictPrint(dict map[int]code) {
	for k, v := range dict {
		fmt.Printf("%d: %v\n", k, v)
	}
}

func treePrint(n *node, depth int) {
	if n == nil {
		return
	}
	fmt.Printf("%s%d\n", strings.Repeat(" ", depth), n.Symbol)
	if depth%2 == 0 {
		treePrint(n.Right, depth+1)
		treePrint(n.Left, depth+1)
	} else {
		treePrint(n.Left, depth+1)
		treePrint(n.Right, depth+1)
	}
}

func codeExists(c code, dict map[code]int) bool {
	_, ok := dict[c]
	return ok
}

func toCode(data []int, dict map[int]code) ([]byte, uint) {
	var bitset *big.Int
	bitset = big.NewInt(0) // todo: set underlying length
	totallen := uint(0)
	for _, sym := range data {
		c := dict[sym]
		bitset.Lsh(bitset, uint(c.len))
		bitset.Or(bitset, big.NewInt(int64(c.code)))
		totallen += uint(c.len)
	}

	bytelen := (totallen + 7) / 8 // round up
	b := bitset.FillBytes(make([]byte, int(bytelen)))
	//fmt.Println(b[:10])
	//fmt.Println("b len", len(b))
	//fmt.Println("totallen", totallen)
	return b, totallen
}

func huffman(data []int) []byte {
	//fmt.Println(maximumAbs(data))

	printEntropy(data)

	probabilities := getProbabilities(data)
	root := getHuffmanTree(probabilities)
	//assertTreeIsFull(root)
	dict := getDictionary(root)

	chunkLen := uint8(math.Ceil(math.Log2(float64(maximumAbs(data) + 1))))
	//chunkLen := uint8(16)
	//fmt.Println("chunklen", chunkLen, "max", maximumAbs(data))
	tree := encodeHuffmanTree2(root, chunkLen)
	//tree := encodeHuffmanTree(root)
	//dictPrint(getDictionary(root))
	code, totallen := toCode(data, dict)

	result := addVarints([]byte{}, uint64(totallen), uint64(chunkLen)) // optimize length later
	result = append(result, tree...)
	result = append(result, code...)
	return result
}

func unHuffman(data []byte) []int {
	totallen, chunklen, tree := get2Varints(data)
	fmt.Println("chunklen", chunklen)
	root, data := decodeHuffmanTree2(tree, uint8(chunklen))
	//root, data := decodeHuffmanTree(tree)
	//dictPrint(getDictionary(root))
	code := fromCode(data, totallen, root)
	return code
}

func printEntropy(data []int) {
	probabilities := getProbabilities(data)
	total := 0
	for _, count := range probabilities {
		total += count
	}
	entropy := 0.0
	for _, count := range probabilities {
		p := float64(count) / float64(total)
		entropy -= p * math.Log2(p)
	}
	fmt.Println("Entropy", entropy)
}

func get1Varint(data []byte) (uint, []byte) {
	totallen, n := binary.Uvarint(data)
	return uint(totallen), data[n:]
}

func encodeHuffmanTree(root *node) []byte {
	result := new(bytes.Buffer)
	ge := gob.NewEncoder(result)
	err := ge.Encode(root)
	if err != nil {
		panic(err)
	}
	fmt.Println("encoded huffman tree len", len(result.Bytes()))
	return result.Bytes()
}

func encodeHuffmanTree2(root *node, chunklen uint8) []byte {
	//treePrint(root, 0)
	type serializedNode struct {
		Symbol   uint64
		Internal bool
	}
	preordered := make([]serializedNode, 0)
	var traverse func(n *node)
	traverse = func(n *node) {
		if n.Symbol != -1 {
			preordered = append(preordered, serializedNode{Symbol: uint64(n.Symbol), Internal: false})
			return
		}
		preordered = append(preordered, serializedNode{Symbol: 0, Internal: true})
		traverse(n.Left)
		traverse(n.Right)
	}
	traverse(root)

	//for i, n := range preordered {
	//	fmt.Println(i, n)
	//}

	bitset := new(big.Int)
	totallen := uint64(len(preordered)) // at least 1 bit per node
	for _, n := range preordered {
		bitset.Lsh(bitset, 1)
		if n.Internal {
			bitset.Or(bitset, big.NewInt(1))
		} else {
			bitset.Or(bitset, big.NewInt(0))

			// add symbol
			bitset.Lsh(bitset, uint(chunklen))
			bitset.Or(bitset, big.NewInt(int64(n.Symbol)))
			totallen += uint64(chunklen)
		}
	}
	bytelen := (totallen + 7) / 8 // round up
	b := bitset.FillBytes(make([]byte, int(bytelen)))

	result := addVarints([]byte{}, totallen)
	result = append(result, b...)
	fmt.Println("encoded huffman tree", len(result))
	return result
}

func decodeHuffmanTree2(data []byte, chunkLen uint8) (*node, []byte) {
	totallen, n := binary.Uvarint(data)
	bitset := new(big.Int)
	bytelen := (totallen + 7) / 8 // round up
	bitset.SetBytes(data[n : n+int(bytelen)])

	type serializedNode struct {
		Symbol   uint64
		Internal bool
	}
	preordered := make([]serializedNode, 0)
	for i := int(totallen - 1); i >= 0; i-- {
		if bitset.Bit(i) == 1 {
			// internal
			preordered = append(preordered, serializedNode{Symbol: 0, Internal: true})
		} else {
			currSymbol := uint64(0)
			for j := uint8(0); j < chunkLen; j++ {
				i--
				currSymbol <<= 1
				currSymbol |= uint64(bitset.Bit(i))
			}
			preordered = append(preordered, serializedNode{Symbol: currSymbol, Internal: false})
		}
	}

	//fmt.Println("decoded huffman tree:")
	//
	//for i, n := range preordered {
	//	fmt.Println(i, n)
	//}

	var root *node
	var traverse func([]serializedNode) *node

	currPos := 0
	traverse = func(nodes []serializedNode) *node {
		n := nodes[currPos]
		if !n.Internal {
			return &node{Symbol: int(n.Symbol)}
		}
		currPos++
		l := traverse(nodes)
		currPos++
		r := traverse(nodes)
		return &node{Left: l, Right: r, Symbol: -1}
	}
	root = traverse(preordered)

	//treePrint(root, 0)
	return root, data[n+int(bytelen):]
}

func assertTreeIsFull(n *node) {
	if n == nil {
		return
	}
	if n.Left == nil {
		if n.Right != nil {
			panic("tree is not full")
		}
	}
	if n.Right == nil {
		if n.Left != nil {
			panic("tree is not full")
		}
	}
	assertTreeIsFull(n.Left)
	assertTreeIsFull(n.Right)
}

func fromCode(data []byte, totallen uint64, root *node) []int {
	var bitset *big.Int
	bitset = new(big.Int)
	bitset.SetBytes(data)

	var result []int
	var n *node
	n = root
	for i := int(totallen - 1); i >= 0; i-- {
		if bitset.Bit(i) == 0 {
			n = n.Left
		} else {
			n = n.Right
		}
		if n.Symbol != -1 {
			result = append(result, n.Symbol)
			n = root
		}
	}
	return result
}

func decodeHuffmanTree(data []byte) (*node, []byte) {
	result := new(bytes.Buffer)
	result.Write(data)
	gd := gob.NewDecoder(result)
	var root *node
	err := gd.Decode(&root)
	if err != nil {
		panic(err)
	}
	// bytes returns the unread portion of the buffer
	return root, result.Bytes()
}
