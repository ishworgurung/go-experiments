package main

type Sleeper interface {
	Sleep()
}

type Cat struct {}

type Dog struct {}

func (c Cat) Sleep() {}

func (d Dog) Sleep() {}

func main() {
	pets := []Sleeper{new(Cat), new(Dog)}
	for _, x := range pets {
		x.Sleep()
	}
}
