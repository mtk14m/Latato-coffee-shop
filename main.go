package main

import "net/http"

func main() {

	println("Go Api...")
	http.ListenAndServe(":9090", nil)
}
