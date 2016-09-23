package main

import (
	"fmt"
	"html"
	"net/http"
	"os"
)

func main() {
	http.HandleFunc("/", helloHandler)
	http.ListenAndServe(":"+os.Getenv("PORT"), nil)
}

func helloHandler(w http.ResponseWriter, r *http.Request) {
	fmt.Fprintf(w, "Hello, %q", html.EscapeString(r.URL.Path))
}
