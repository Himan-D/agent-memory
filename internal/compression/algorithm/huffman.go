package algorithm

import (
	"container/heap"
	"errors"
	"fmt"
	"strings"
)

type HuffmanNode struct {
	Char  string
	Freq  int
	Left  *HuffmanNode
	Right *HuffmanNode
}

type HuffmanTree struct {
	codes   map[string]string
	reverse map[string]string
	root    *HuffmanNode
}

type huffmanItem struct {
	node *HuffmanNode
	idx  int
}

type huffmanHeap []huffmanItem

func (h huffmanHeap) Len() int           { return len(h) }
func (h huffmanHeap) Less(i, j int) bool { return h[i].node.Freq < h[j].node.Freq }
func (h huffmanHeap) Swap(i, j int)      { h[i], h[j] = h[j], h[i] }

func (h *huffmanHeap) Push(x any) {
	*h = append(*h, x.(huffmanItem))
}

func (h *huffmanHeap) Pop() any {
	n := len(*h)
	item := (*h)[n-1]
	*h = (*h)[:n-1]
	return item
}

func (n *HuffmanNode) isLeaf() bool {
	return n.Left == nil && n.Right == nil
}

func BuildHuffmanTree(frequencies map[string]int) (*HuffmanNode, error) {
	if len(frequencies) == 0 {
		return nil, errors.New("no frequencies provided")
	}

	h := &huffmanHeap{}
	idx := 0
	for char, freq := range frequencies {
		if freq > 0 {
			heap.Push(h, huffmanItem{
				node: &HuffmanNode{Char: char, Freq: freq},
				idx:  idx,
			})
			idx++
		}
	}

	if h.Len() == 0 {
		return nil, errors.New("no valid frequencies")
	}

	for h.Len() > 1 {
		item1 := heap.Pop(h).(huffmanItem)
		item2 := heap.Pop(h).(huffmanItem)

		parent := &HuffmanNode{
			Char: "",
			Freq:  item1.node.Freq + item2.node.Freq,
			Left:  item1.node,
			Right: item2.node,
		}
		heap.Push(h, huffmanItem{node: parent, idx: idx})
		idx++
	}

	item := heap.Pop(h).(huffmanItem)
	return item.node, nil
}

func GenerateCodes(node *HuffmanNode, prefix string, codes map[string]string) {
	if node == nil {
		return
	}

	if node.isLeaf() {
		if node.Char != "" {
			if prefix == "" {
				prefix = "0"
			}
			codes[node.Char] = prefix
		}
		return
	}

	GenerateCodes(node.Left, prefix+"0", codes)
	GenerateCodes(node.Right, prefix+"1", codes)
}

func NewHuffmanTree(text string) (*HuffmanTree, error) {
	freqs := make(map[string]int)
	for _, ch := range text {
		freqs[string(ch)]++
	}

	root, err := BuildHuffmanTree(freqs)
	if err != nil {
		return nil, err
	}

	codes := make(map[string]string)
	GenerateCodes(root, "", codes)

	reverse := make(map[string]string)
	for k, v := range codes {
		reverse[v] = k
	}

	return &HuffmanTree{
		codes:   codes,
		reverse: reverse,
		root:    root,
	}, nil
}

func (t *HuffmanTree) Encode(text string) (string, error) {
	var encoded strings.Builder
	for _, ch := range text {
		code, ok := t.codes[string(ch)]
		if !ok {
			return "", fmt.Errorf("character '%c' not in tree", ch)
		}
		encoded.WriteString(code)
	}
	return encoded.String(), nil
}

func (t *HuffmanTree) Decode(bits string) (string, error) {
	if t.root == nil {
		return "", errors.New("empty tree")
	}

	var decoded strings.Builder
	node := t.root

	for i := 0; i < len(bits); i++ {
		if node.isLeaf() {
			if node.Char != "" {
				decoded.WriteString(node.Char)
			}
			node = t.root
		}

		if node == nil {
			continue
		}

		switch bits[i] {
		case '0':
			node = node.Left
		case '1':
			node = node.Right
		}
	}

	if node != nil && node.isLeaf() && node.Char != "" {
		decoded.WriteString(node.Char)
	}

	return decoded.String(), nil
}

func (t *HuffmanTree) CompressionRatio(original string, encoded string) float64 {
	if len(original) == 0 {
		return 0
	}
	bitsPerChar := 16.0
	return 1.0 - (float64(len(encoded)) / float64(len(original))) * bitsPerChar / 100
}

func (t *HuffmanTree) Codes() map[string]string {
	return t.codes
}

func (t *HuffmanTree) ReverseMap() map[string]string {
	return t.reverse
}

func (t *HuffmanTree) Root() *HuffmanNode {
	return t.root
}

func (n *HuffmanNode) IsLeaf() bool {
	return n.Left == nil && n.Right == nil
}