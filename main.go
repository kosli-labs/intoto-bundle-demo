// A deliberately small web server: the demo is the pipeline, not the app.
package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintln(w, "Hello from intoto-bundle-demo")
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
