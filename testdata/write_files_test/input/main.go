package main

import "os"

func main() {
	foo(os.Stdout)
}

func init() {
	println("I am in init!")
}
