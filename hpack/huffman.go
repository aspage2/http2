package hpack

import (
	"bufio"
	"bytes"
	"fmt"
	"io"
	_ "embed"
)

//go:embed huffmantrimmed.txt
var hpackHuffmanCode []byte

var HpackHuffmanTree *HuffmanTree

func init() {
	var err error
	HpackHuffmanTree, err = treeFromReader(bytes.NewReader(hpackHuffmanCode))
	if err != nil {
		panic(err)
	}
}

const EOS uint16 = 256

type HuffmanTree struct {
	root    *TrieNode
	jumpMap [257]*TrieNode
}

func NewHuffmanTree() *HuffmanTree {
	var t HuffmanTree
	t.root = new(TrieNode)
	return &t
}

func (ht *HuffmanTree) Insert(seq []bool, sym uint16) {
	ht.jumpMap[sym] = ht.root.Insert(seq, sym)
}

func (ht *HuffmanTree) Decode(input []uint8) []uint8 {
	var ret []uint8
	curr := ht.root
	for byteNo := 0; byteNo < len(input); byteNo++ {
		for i := 7; i >= 0; i-- {
			left := (input[byteNo]>>i)&0x1 != 0
			next := curr.Child(left)
			if next.Left == nil && next.Right == nil {
				if next.Sym == EOS {
					fmt.Println("FOUND EOS")
				}
				ret = append(ret, uint8(next.Sym))
				curr = ht.root
			} else {
				curr = next
			}
		}
	}
	return ret
}

func (ht *HuffmanTree) Encode(data []uint8) []uint8 {
	var (
		ret    []uint8
		byteNo = 0
		bitNo  = 7
	)
	ret = append(ret, 0)
	for _, sym := range data {
		var vals []bool
		n := ht.jumpMap[sym]
		for n != ht.root {
			vals = append(vals, n == n.Parent.Left)
			n = n.Parent
		}

		for i := len(vals) - 1; i >= 0; i-- {
			v := vals[i]
			if v {
				ret[byteNo] |= 1 << bitNo
			}
			if bitNo == 0 {
				byteNo += 1
				bitNo = 7
				ret = append(ret, 0)
			} else {
				bitNo--
			}
		}
	}
	if bitNo == 7 {
		ret = ret[:len(ret)-1]
	} else {
		for i := 0; i <= bitNo; i++ {
			ret[byteNo] |= 1 << i
		}
	}
	return ret
}

type TrieNode struct {
	Parent *TrieNode
	Left   *TrieNode
	Right  *TrieNode
	Sym    uint16
}

func (tn *TrieNode) Insert(seq []bool, sym uint16) *TrieNode {
	if len(seq) == 0 {
		tn.Sym = sym
		return tn
	}
	return tn.Child(seq[0]).Insert(seq[1:], sym)
}

func (tn *TrieNode) Child(left bool) *TrieNode {
	var ret **TrieNode
	if left {
		ret = &tn.Left
	} else {
		ret = &tn.Right
	}

	if *ret == nil {
		*ret = new(TrieNode)
		(*ret).Parent = tn
	}

	return *ret
}

func treeFromReader(data io.Reader) (*HuffmanTree, error) {
	rd := bufio.NewScanner(data)
	t := NewHuffmanTree()

	var sym uint16
	for rd.Scan() {
		line := rd.Text()
		seq := make([]bool, len(line))
		for i, c := range line {
			seq[i] = c == '1'
		}
		t.Insert(seq, sym)
		sym += 1
	}
	if err := rd.Err(); err != nil {
		return nil, err
	}
	return t, nil
}
