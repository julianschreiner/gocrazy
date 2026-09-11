package main

import "todocli/persistance"

func main() {
	action := parseCliFlags()

	repo := persistance.NewJSONRepository("./storage.txt")
}

func parseCliFlags() any {
	panic("unimplemented")
}
