package main

import (
	"fmt"
	"math/rand"
	"sort"
	"time"
)

func intersection(a []int, b []int) (inter []int) {
	// interacting on the smallest list first can potentailly be faster...but not by much, worse case is the same
	low, high := a, b
	if len(a) > len(b) {
		low = b
		high = a
	}

	done := false
	for i, l := range low {
		for j, h := range high {
			// get future index values
			f1 := i + 1
			f2 := j + 1
			if l == h {
				inter = append(inter, h)
				if f1 < len(low) && f2 < len(high) {
					// if the future values aren't the same then that's the end of the intersection
					if low[f1] != high[f2] {
						done = true
					}
				}
				// we don't want to interate on the entire list everytime, so remove the parts we already looped on will make it faster each pass
				high = high[:j+copy(high[j:], high[j+1:])]
				break
			}
		}
		// nothing in the future so we are done
		if done {
			break
		}
	}
	return
}

func main() {
	// slice1 := []string{"foo", "bar", "hello", "bar"}
	// slice2 := []string{"foo", "bar"}
	// fmt.Printf("%+v\n", intersection(slice1, slice2))
	rand.Seed(time.Now().UnixNano() + time.Now().UnixNano())
	var o1, o2 []int
	var r1 = 1000     // rand.Intn(9100)
	var r2 = 10000000 //9100 //rand.Intn(50000000)
	for i := 0; i <= r1; i++ {
		o1 = append(o1, i)
	}
	for z := r2; z > 0; z-- {
		o2 = append(o2, z)
	}
	s1 := time.Now()
	l1 := intersection(o1, o2)
	fmt.Printf("l=%v, e=%v\n", len(l1), time.Now().Sub(s1))
	sort.Ints(l1)
	fmt.Printf("intersected len = %d\n", len(l1))
	fmt.Printf("intersected data = %v\n", l1)
	fmt.Printf("r1=%v\nr2=%v\n", r1, r2)
}
