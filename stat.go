// Copyright 2016 The ql Authors. All rights reserved.
// Use of this source code is governed by a BSD-style
// license that can be found in the LICENSE file.

// +build ignore

package main

import (
	"bufio"
	"fmt"
	"log"
	"os"
	"sort"
	"strconv"
	"strings"

	"github.com/cznic/mathutil"
)

func main() {
	var nallocs, allocs, minAlloc, maxAlloc int
	minAlloc = mathutil.MaxInt
	allocMap := map[int]int{}

	var nfree, frees, minFree, maxFree int
	minFree = mathutil.MaxInt
	freeMap := map[int]int{}

	log.SetFlags(log.Lshortfile)
	s := bufio.NewScanner(os.Stdin)
	var v []int
	for s.Scan() {
		line := s.Text()
		a := strings.Fields(line)
		if len(a) == 0 {
			log.Panic()
		}

		v = v[:0]
		for _, f := range a {
			n, err := strconv.ParseUint(f, 10, 31)
			if err != nil {
				v = append(v, -1)
				continue
			}

			v = append(v, int(n))
		}

		switch a[0] {
		case "alloc":
			n := v[1]
			if n < 0 {
				log.Panic()
			}

			nallocs++
			allocs += n
			minAlloc = mathutil.Min(minAlloc, n)
			maxAlloc = mathutil.Max(maxAlloc, n)
			allocMap[n]++
		case "free":
			n := v[1]
			if n < 0 {
				log.Panic()
			}

			nfree++
			frees += n
			minFree = mathutil.Min(minFree, n)
			maxFree = mathutil.Max(maxFree, n)
			freeMap[n]++
		default:
			//log.Panic(a[0])
		}
	}
	if err := s.Err(); err != nil {
		log.Fatal(err)
	}

	fmt.Printf("allocs %d, total %d, min %d, avg %d, max %d\n", nallocs, allocs, minAlloc, allocs/nallocs, maxAlloc)
	cum := .0
	var a []int
	for k := range allocMap {
		a = append(a, k)
	}
	sort.Ints(a)
	for _, k := range a {
		s := 100 * float64(allocMap[k]) / float64(nallocs)
		cum += s
		fmt.Printf("%4d: %5d %8.2f%% %8.2f%%\n", k, allocMap[k], s, cum)
	}

	fmt.Printf("frees %d, total %d, min %d, avg %d, max %d\n", nfree, frees, minFree, frees/nfree, maxFree)
	cum = .0
	a = a[:0]
	for k := range freeMap {
		a = append(a, k)
	}
	sort.Ints(a)
	for _, k := range a {
		s := 100 * float64(freeMap[k]) / float64(nfree)
		cum += s
		fmt.Printf("%4d: %5d %8.2f%% %8.2f%%\n", k, freeMap[k], s, cum)
	}
}
