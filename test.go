package main

import (
	"fmt"
	"time"
)

func main() {
	orderTime := time.Unix(1725911912, 0)
	diff := time.Since(orderTime).Minutes()
	fmt.Println(orderTime)
	fmt.Println(diff)

}
