package main

import (
	"fmt"
	"sort"
)


type user struct {
	name string
	age  int
}

func main() {
	list := []user{
		user{name: "A", age: 2},
		user{name: "B", age: 1},
		user{name: "C", age: 4},
		user{name: "D", age: 3},
	}

	list2 := []user{
		user{name: "A", age: 2},
		user{name: "B", age: 1},
		user{name: "C", age: 4},
		user{name: "D", age: 3},
	}

	sort.Slice(list, func(i, j int) bool {
		return list[i].age < list[j].age
	})

	fmt.Println(list)
	fmt.Println(list2)
}