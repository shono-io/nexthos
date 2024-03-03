package main

import "time"

func main() {
	i := 0
	for i < 100 {
		println("#%d Hello, World!", i)
		i++
		time.Sleep(1 * time.Second)
	}
}
