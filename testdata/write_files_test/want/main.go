package main

import "os"

func init() {
	println("I am in init!")
}

func main() {
	foo(os.Stdout)
}
