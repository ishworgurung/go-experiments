package main

import (
	"fmt"
	"math"
	"testing"

	mapset "github.com/deckarep/golang-set"
)

//func BenchmarkMyIntersect(b *testing.B) {
//	//rand.Seed(time.Now().UnixNano() + time.Now().UnixNano())
//	var r1 = 1000     //rand.Intn(10)
//	var r2 = 10000000 //rand.Intn(10000000)
//	var o1, o2 []int
//	for i := 0; i <= r1; i++ {
//		o1 = append(o1, i)
//	}
//	for z := r2; z > 0; z-- {
//		o2 = append(o2, z)
//	}
//	var xx []int
//	for n := 0; n < b.N; n++ {
//		xx = myIntersect(r1, r2)
//	}
//	resultIntSlice = xx
//	//s1 := time.Now()
//	//l1 := smartIntersect(o1, o2)
//	//fmt.Printf("\nmyIntersect()\n===========================\n")
//	//fmt.Printf("e=%v\n", time.Now().Sub(s1))
//	//fmt.Printf("intersected len = %d\n", len(l1))
//	//sort.Ints(l1)
//	//fmt.Printf("intersected data = %v\n", l1)
//	//fmt.Printf("r1=%v\nr2=%v\no1=%v\no2=%v\n", r1, r2, len(o1), len(o2))
//}

func BenchmarkFastIntersect(b *testing.B) {
	//rand.Seed(time.Now().UnixNano() + time.Now().UnixNano())
	var r1 = 1000     // rand.Intn(10)
	var r2 = 10000000 //rand.Intn(10000000)
	o1 := mapset.NewSet()
	o2 := mapset.NewSet()
	for i := 0; i <= r1; i++ {
		o1.Add(i)
	}
	for z := r2; z > 0; z-- {
		o2.Add(z)
	}
	b.ResetTimer()
	for k := 0.; k <= float64(r1); k++ {
		n := int(math.Pow(2, k))
		b.Run(fmt.Sprintf("%.2f/%d", k, n), func(b *testing.B) {
			for i := 0; i < b.N; i++ {
				fastIntersect(int(n), r2)
			}
		})
	}
	//s1 := time.Now()
	//l1 := o1.Intersect(o2)
	//fmt.Printf("\nfastIntersect()\n===========================\n")
	//fmt.Printf("e=%v\n", time.Now().Sub(s1))
	//fmt.Printf("intersected len = %d\n", l1.Cardinality())
	//fmt.Printf("intersected data = %v\n", l1)
	//fmt.Printf("r1=%v\nr2=%v\no1=%v\no2=%v\n", r1, r2, o1.Cardinality(), o2.Cardinality())
}
