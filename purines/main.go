package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"
)

func init() {
	log.SetFlags(0)
}

// purines is used to encode i, 'A', 'T', 'G', 'C' stands for 0B00, 0B01, 0B10, 0B11 respectively while encoding
// ell is the length of the final encoded strings
// cgLowBound and cgHighBound are the bonds on C+G, here they are 12 and 14 respectively
// checkMark and ballotX are symbols to indicate valid or invalid combinations
const purines = "ATGC"
const ell = 20
const cgLowBound = 12
const cgHighBound = 14

// numberGoroutines represents the number of goroutines launched while processing data
// numberBatchs represents the number of times to find all the valid strings, each time a new file is generated to store the results
const numberGoroutines = 10
const numberBatchs = 96

// patternInts stands for "AAAA", "TTTT", "GGGG", "CCCC" respectively, in the form of a digit slice (0B0000 0000, 0B0101 0101, 0B1010 1010, 0B1111 1111)
// patternStrings store the patterns to match
// wg is used to monitor goroutines
var (
	patternInts    = []int{0x00, 0x55, 0xaa, 0xff}
	patternStrings = []string{"AAAA", "TTTT", "GGGG", "CCCC"}
	wg             sync.WaitGroup
)

// Pair is used to store the pair <validStr, cgCnt>
type Pair struct {
	str   string
	cgCnt int
}

// subsets is responsible for  finding all valid strings of length 10 with the following constraints:
// 1) it contains no "AAAA", "TTTT", "GGGG", "CCCC" substrings
// 2) the number of character 'C' and 'G' should be above 2
// results are stored into a Pair slice, where each item stores a valid string and the number of 'C'+'G'
func subsets() []Pair {
	res := make([]Pair, 0)
	length := ell / 2
	N := 1 << uint(2*length)

	for i := 0; i < N; i++ {
		// to store encoded results of i
		str := make([]byte, length)
		// count the numbers of 'C' and 'G'
		cgCnt := 0
		// match = 0B11, used to extract the character ('A', 'T', 'G', 'C'), for counting purpose
		matchChar := 0x3
		// matchPattern = 0B11111111, used to extract consequent binary of length 8, e.g., "0101010101",
		// for further patterns match (0x00, 0x55, 0xaa, 0xff)
		matchPattern := 0xff
		// to indicate whether i is valid or not
		flag := true

		for j := 0; j < length; j++ {
			//encode i into a quaternary number represented by 'A', 'T', 'G', and 'C' (0, 1, 2, 3 respectively)
			num := (matchChar & i) >> uint(2*j)
			str[length-1-j] = purines[num]

			// 'C'+'G' check
			if num == 2 || num == 3 {
				cgCnt++
			}
			if (j + 1 - cgCnt) > (ell - cgLowBound) {
				flag = false
				break
			}

			// "AAAA", "TTTT", "GGGG", "CCCC" check
			if j >= 3 {
				for _, patternInt := range patternInts {
					// extract 8-bit number and use XOR to check whether it's a pattern
					if ((i&matchPattern)>>uint((2*j-6)))^patternInt == 0 {
						flag = false
						break
					}
				}
				if !flag {
					break
				}

				// shift left to match the next character
				matchPattern <<= 2
			}

			// shift left to match the next pattern
			matchChar <<= 2
		}

		// find one valid string, store it into the result slice
		if flag {
			res = append(res, Pair{string(str), cgCnt})
		}
	}

	return res
}

// merge is responsible for find all valid combinations
// it merge two strings of length 10 in to a string of length 20
// and find all valid ones fulfilling the following constraints:
// 1) it contains no "AAAA", "TTTT", "GGGG", "CCCC" substrings
// 2) the number of character 'C' and 'G' should be in the range [12, 14]
// final results are written in to a given file
func merge(src []Pair, start, end int, file *os.File, ch chan<- int) {
	defer wg.Done()

	// number of matches for this subroutine
	numberMatches := 0
	// concanated string
	var strMerged string

	for i := start; i < end; i++ {
		// store all valid strings
		// apply for a maximum possible storage to avoid reallocation during growth
		matchStrings := make([]string, 0, len(src))
		for j := 0; j < len(src); j++ {
			// to indicate a valid string or not
			flag := true

			// 'C'+'G' check
			cgTotal := src[i].cgCnt + src[j].cgCnt
			if cgTotal < cgLowBound || cgTotal > cgHighBound {
				continue
			}

			// "AAAA", "TTTT", "GGGG", "CCCC" check
			strMerged = src[i].str + src[j].str
			for _, substr := range patternStrings {
				// check only whether pattern strings exist in the inner substring of length 6
				if strings.Contains(strMerged[7:13], substr) {
					flag = false
					break
				}
			}
			if !flag {
				continue
			}

			// increase the counter and store this valid string
			numberMatches++
			matchStrings = append(matchStrings, strMerged)
		}

		// write after every inner loop finishes
		// use a string builder to build the whole results into a whole string first to lessen IO times
		var b strings.Builder
		for _, str := range matchStrings {
			b.WriteString(str)
			b.WriteString("\n")
		}
		file.WriteString(b.String())
	}

	// return the number of total valid strings for this subroutine
	ch <- numberMatches
}

// This program is used to find all strings of length 20 consisting of 'A', 'T', 'G', 'C'
// each valid one should fulfill the following requirements:
// 1) the number of 'C' and 'G' is no less than 12 and no more than 14
// 2) it contains no substrings like "AAAA", "TTTT", "GGGG", and "CCCC"

// Algorithm: divide and conquer
// 1. find all valid strings of length 10 (using encoding techniques)
// 2. find all valid strings of length 20 by combinations of strings of length 10
// 3. multiple goroutines are launched in this program to take the advantage of  multi-core processor
// 4. results are written into multiple files (199114775296 pairs, about 4TB)
func main() {
	timeProgramStart := time.Now()
	log.Printf("The program starts at %v\n", timeProgramStart)

	src := subsets()
	batchSize := len(src) / numberBatchs
	numberTotalMatches := 0

	for i := 0; i <= numberBatchs; i++ {
		numberLocalMatches := 0

		// open a new file to store the results for this batch
		resultFileName := fmt.Sprintf("result%d.txt", i)
		resultFile, err := os.OpenFile(resultFileName, os.O_WRONLY|os.O_APPEND|os.O_CREATE, 0666)
		if err != nil {
			log.Println("open result file failed!", err.Error())
			os.Exit(1)
		}
		defer resultFile.Close()

		batchStart := i * batchSize
		batchEnd := (i + 1) * batchSize
		if i == numberBatchs {
			batchEnd = len(src)
		}

		// launch multiple goroutines to process the data
		// monitor goroutines and record the number of matches for every goroutine
		loadPerGoroutine := (batchEnd - batchStart) / numberGoroutines
		wg.Add(numberGoroutines + 1)
		ch := make(chan int)

		log.Printf("\nbatch %d begins, batch size: %d, running goroutines: %d\n", i, batchEnd-batchStart, numberGoroutines)
		timeBatchBegin := time.Now()
		for j := 0; j <= numberGoroutines; j++ {
			leftBond := batchStart + j*loadPerGoroutine
			rightBond := batchStart + (j+1)*loadPerGoroutine
			if j == numberGoroutines {
				rightBond = batchEnd
			}

			go merge(src, leftBond, rightBond, resultFile, ch)
		}

		go func() {
			wg.Wait()
			close(ch)
		}()

		for cnt := range ch {
			numberLocalMatches += cnt
		}

		timeBatchElapsed := time.Since(timeBatchBegin)
		log.Printf("batch %d finished.\n", i)
		log.Printf("Time used for this batch: %v, found matches: %d\n", timeBatchElapsed, numberLocalMatches)

		numberTotalMatches += numberLocalMatches
	}

	timeProgramElapsed := time.Since(timeProgramStart)
	log.Printf("\n%d batchs processed, program finished!\n", numberBatchs)
	log.Printf("Total time used: %v, total matches: %d\n", timeProgramElapsed, numberTotalMatches)
}
