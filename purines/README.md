# Project purines

This program is used to find all strings of length 20 consisting of 'A', 'T', 'G', 'C', each valid one should satisfy the following requirements:

1. the number of 'C' and 'G' is no less than 12 and no more than 14
2. it contains no substrings like "AAAA", "TTTT", "GGGG", and "CCCC"

Algorithm: divide and conquer

1. find all valid strings of length 10 (using encoding techniques)
2. find all combinations of two strings of length 10, and filter out invalid ones

Final results: 199114775296 pairs, about 4TB in storage.
