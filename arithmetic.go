package main

import (
	"fmt"
	"math/big"
	"os"
)

var radix = 100000

func getCounts(data []int) map[int]int {
	counts := make(map[int]int)
	for _, sym := range data {
		counts[sym]++
	}
	return counts
}

func cumulativeFreqs(counts map[int]int) map[int]int {
	freqs := make(map[int]int)
	sum := 0
	for sym, count := range counts {
		freqs[sym] = sum
		sum += count
	}
	return freqs
}

func arithEncode(data []int) []byte {
	counts := getCounts(data)
	cf := cumulativeFreqs(counts)

	bigBase := big.NewInt(int64(len(data)))
	bigLow := big.NewInt(0)

	fp := big.NewInt(1) // frequency product

	for _, sym := range data {
		c := big.NewInt(int64(cf[sym]))

		bigLow.Mul(bigLow, bigBase)
		bigLow.Add(bigLow, c.Mul(c, fp))
		fp.Mul(fp, big.NewInt(int64(counts[sym])))
	}

	bigHigh := big.NewInt(0).Add(bigLow, fp)

	bigOne := big.NewInt(1)
	bigZero := big.NewInt(0)
	bigRadix := big.NewInt(int64(radix))

	tmp := big.NewInt(0).Set(fp)
	powr := big.NewInt(0)

	for {
		tmp.Div(tmp, bigRadix)
		if tmp.Cmp(bigZero) == 0 {
			break
		}
		powr.Add(powr, bigOne)
	}

	diff := big.NewInt(0)
	diff.Sub(bigHigh, bigOne)
	diff.Div(diff, big.NewInt(0).Exp(bigRadix, powr, nil))

	fmt.Println(len(diff.Bytes()), powr, counts)

	saveBytes(diff.Bytes())

	return nil
}

func saveBytes(data []byte) {
	filename := "saved.bin"
	f, err := os.Create(filename)
	if err != nil {
		panic(err)
	}
	defer f.Close()
	f.Write(data)
}
