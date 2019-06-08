package main

import (
	"fmt"
	"sync"
	"time"
)

// purines is used to encode i, 'A', 'T', 'G', 'C' stands for 0B00, 0B01, 0B10, 0B11 respectively
// ell is the length of the final encoded strings
// lowBound and highBound are the bonds on C+G, here they are 12 and 14 respectively
// checkMark and ballotX are symbols to indicate valid or invalid combinations
const purines = "ATGC"
const ell = 20
const lowBound = 12
const highBound = 14
const checkMark = "\u2713"
const ballotX = "\u2717"

// patterns stands for "AAAA", "TTTT", "GGGG", "CCCC" respectively, in the form of a digit slice (0B0000 0000, 0B0101 0101, 0B1010 1010, 0B1111 1111)
// wg is used to monitor go routines
var (
	patterns = []int{0x00, 0x55, 0xaa, 0xff}
	wg       sync.WaitGroup
)

// subsets is the main function for calculating matched patterns
// start and end stand for the interval [start, end)
func subsets(start, end int, ch chan<- int) {
	defer wg.Done()
	total := 0

	for i := start; i < end; i++ {
		// match = 0B11, used to extract the character ('A', 'T', 'G', 'C'), for counting purpose
		match := 0x3
		// matchPattern = 0B11111111, used to extract consequent four characters (e.g., "ATTG", to match patterns ("AAAA", "TTTT", "GGGG", "CCCC")
		matchPattern := 0xff
		// to indicate whether i is valid or not
		flag := true
		// count the numbers of 'C' and 'G'
		cgCnt := 0
		// to store encoded results of i
		str := make([]byte, ell)

		for j := 0; j < ell; j++ {
			//encode i into a string consisting of  'A', 'T', 'G', and 'C'
			num := (match & i) >> uint(2*j)
			str[ell-1-j] = purines[num]

			// C+G judgement
			// high bound
			if num == 2 || num == 3 {
				cgCnt++
				if cgCnt > highBound {
					flag = false
					break
				}
			}
			// low bound
			if (j + 1 - cgCnt) > (ell - lowBound) {
				flag = false
				break
			}

			// "AAAA", "TTTT", "GGGG", "CCCC" judgement
			if j >= 3 {
				for _, pattern := range patterns {
					if ((i&matchPattern)>>uint((2*j-6)))^pattern == 0 {
						flag = false
						break
					}
				}
				if !flag {
					break
				}
				matchPattern <<= 2
			}

			match <<= 2
		}

		if flag {
			total++
			fmt.Println(string(str), checkMark, cgCnt)
		}
	}

	ch <- total
}

/*
This program is used to find all strings consists of 'A', 'T', 'G', 'C' of length 20 which satisfy the following requirements:
1) the number of 'C' and 'G' is no less than 12 and no more than 14
2) it contains no substrings like "AAAA", "TTTT", "GGGG", and "CCCC"
*/
func main() {
	t1 := time.Now()
	// start := 0x0101ababab
	// end := 0x0101ababab + 100
	start := 0
	end := 1 << uint(2*ell)
	total := 0

	numberGoRoutines := 100
	taskLoad := (end - start) / numberGoRoutines

	wg.Add(numberGoRoutines + 1)
	ch := make(chan int, 100)

	for i := 0; i < numberGoRoutines; i++ {
		l := start + i*taskLoad
		r := start + (i+1)*taskLoad
		go subsets(l, r, ch)
	}
	go subsets(start+numberGoRoutines*taskLoad, end, ch)

	go func() {
		wg.Wait()
		close(ch)
	}()

	for cnt := range ch {
		total += cnt
	}

	elapsed := time.Since(t1)
	fmt.Printf("total = %d\n", total)
	fmt.Printf("[%d, %d), %d number checked with %d goroutines, time elapsed: %v.\n", start, end, end-start, numberGoRoutines, elapsed)
}
