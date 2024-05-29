package main

var EndSymbol = -1 // the diffs get shifted to be non-negative so this is fine.

//func init() {
//	//fmt.Println(bwt([]int{1, 2, 2, 3, 4, 4, 3, 3}))
//	//fmt.Println(bwt([]int{2, 1, 10, 1, 10, 1}))
//}

func mtf(data []int) []int {
	table := make([]int, 1025)
	for i := 0; i < 1025; i++ {
		table[i] = i
	}
	table[1024] = EndSymbol
	result := make([]int, len(data))
	for i, sym := range data {
		for j, t := range table {
			if t == sym {
				result[i] = j
				copy(table[1:], table[:j])
				table[0] = sym
				break
			}
		}
	}
	return result
}

func bwt(data []int) []int {
	data = append(data, EndSymbol)
	n := len(data)
	sa := NewSuffixArrayX(data)
	//fmt.Println("index", sa.index)
	bwt := make([]int, n)
	for i := 0; i < n; i++ {
		if sa.index[i] == 0 {
			bwt[i] = EndSymbol
		} else {
			bwt[i] = data[sa.index[i]-1]
		}
	}
	return bwt //, sa.index[0]
}

// adapted from https://github.com/cweill/SuffixArray-Golang/blob/master/suffixarrayx.go
type suffixarrayx struct {
	CUTOFF int
	data   []int
	index  []int
	n      int
}

// Constructor
func NewSuffixArrayX(data []int) *suffixarrayx {
	sa := new(suffixarrayx)
	sa.n = len(data)
	//data = str + "\n"
	sa.CUTOFF = 5
	sa.data = data
	sa.index = make([]int, sa.n)

	for i := 0; i < sa.n; i++ {
		sa.index[i] = i
	}
	// shuffle
	sa.sort(0, sa.n-1, 0)
	return sa
}

// 3-way string quicksort lo..hi starting at dth character
func (sa *suffixarrayx) sort(lo, hi, d int) {
	// cutoff to insertion sort for small subarrays
	if hi <= lo+sa.CUTOFF {
		sa.insertion(lo, hi, d)
		return
	}
	lt, gt := lo, hi
	v := sa.data[sa.index[lo]+d]
	i := lo + 1
	for i <= gt {
		t := sa.data[sa.index[i]+d]
		if t < v {
			sa.exch(lt, i)
			lt++
			i++
		} else if t > v {
			sa.exch(i, gt)
			gt--
		} else {
			i++
		}
	}

	// a[lo..lt-1] < v = a[lt..gt] < a[gt+1..hi].
	sa.sort(lo, lt-1, d)
	if v > 0 {
		sa.sort(lt, gt, d+1)
	}
	sa.sort(gt+1, hi, d)
}

// sort from a[lo] to a[hi], starting at the dth character
func (sa *suffixarrayx) insertion(lo, hi, d int) {
	for i := lo; i <= hi; i++ {
		for j := i; j > lo && sa.less(sa.index[j], sa.index[j-1], d); j-- {
			sa.exch(j, j-1)
		}
	}
}

// is data[i+d..N) < data[j+d..N) ?
func (sa *suffixarrayx) less(i, j, d int) bool {
	if i == j {
		return false
	}
	i = i + d
	j = j + d
	for i < sa.n && j < sa.n {
		if sa.data[i] < sa.data[j] {
			return true
		}
		if sa.data[i] > sa.data[j] {
			return false
		}
		i++
		j++
	}
	return i > j
}

// exchange index[i] and index[j]
func (sa *suffixarrayx) exch(i, j int) {
	swap := sa.index[i]
	sa.index[i] = sa.index[j]
	sa.index[j] = swap
}
