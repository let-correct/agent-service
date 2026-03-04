package main

import (
	"fmt"
	"log"
	"net/http"
)

func main() {
	http.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		code := r.URL.Query().Get("code")
		state := r.URL.Query().Get("state")

		fmt.Printf("\n--- OAuth Callback Received ---\ncode:  %s\nstate: %s\n-------------------------------\n\n", code, state)

		w.Header().Set("Content-Type", "text/html")
		fmt.Fprintf(w, `<html><body>
<h2>Callback received</h2>
<p><strong>code:</strong> %s</p>
<p><strong>state:</strong> %s</p>
<p>Check your terminal, then POST these to your Lambda callback endpoint.</p>
</body></html>`, code, state)
	})

	log.Println("listening on http://localhost:3000")
	log.Fatal(http.ListenAndServe(":3000", nil))
}
