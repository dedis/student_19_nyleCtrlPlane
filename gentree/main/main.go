package main

import "github.com/dedis/paper_crux/dsn_exp/gentree"

func main() {
	minTreeIndex := len(gentree.GenerateRadius(10000.0)) - 1
	println(minTreeIndex)
}
