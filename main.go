package main

import (
	"bytes"
	"context"
	_ "embed"
	"encoding/json"
	"fmt"
	"github.com/rs/cors"
	"math/rand"
	"net/http"
	"strings"
	"time"
)

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/random-coord", func(w http.ResponseWriter, r *http.Request) {
		flusher, ok := w.(http.Flusher)
		if !ok {
			http.Error(w, "SSE not supported", http.StatusInternalServerError)
			return
		}

		fmt.Println("Request received for coords...")

		w.Header().Set("Content-Type", "text/event-stream")

		priceCh := make(chan string)

		// Start a go routine that will send price updates the on the price channel.
		// These price updates will be sent back to the client.
		go generateCoord(r.Context(), priceCh)

		for price := range priceCh {
			event, err := formatServerSentEvent("message", price)
			if err != nil {
				fmt.Println(err)
				break
			}

			_, err = fmt.Fprint(w, event)
			if err != nil {
				fmt.Println(err)
				break
			}

			flusher.Flush()
		}

		fmt.Println("Finished sending coords...")
	})

	handler := cors.Default().Handler(mux)
	http.ListenAndServe(":4444", handler)
}

// generateCoord generates price as random integer and sends it the
// provided channel every 1 second.
func generateCoord(ctx context.Context, priceCh chan<- string) {
	r := rand.New(rand.NewSource(time.Now().Unix()))

	ticker := time.NewTicker(time.Second * 10)
	coords := [5]string{
		"-16.499586146571012,-68.159141379932314",
		"-16.48905192924404,-68.08858855399531",
		"-16.48131549825551,-68.17579253349894",
		"-16.536615948734706,-68.16703780327317",
		"-16.54467927520438,-68.06421263846475",
	}

outerloop:
	for {
		select {
		case <-ctx.Done():
			break outerloop
		case <-ticker.C:
			random_index := r.Intn(5)
			priceCh <- coords[random_index]
		}
	}

	ticker.Stop()

	close(priceCh)

	fmt.Println("Finished generating coords")
}

// formatServerSentEvent takes name of an event and any kind of data and transforms
// into a server sent event payload structure.
// Data is sent as a json object, { "data": <your_data> }.
//
// Example:
//
//	Input:
//		event="price-update"
//		data=10
//	Output:
//		event: price-update\n
//		data: "{\"data\":10}"\n\n
func formatServerSentEvent(event string, coords any) (string, error) {
	coods_str := coords.(string)
	coords_array := strings.Split(coods_str, ",")
	object := map[string]any{
		"lat":  coords_array[0],
		"long": coords_array[1],
	}

	buff := bytes.NewBuffer([]byte{})

	encoder := json.NewEncoder(buff)

	err := encoder.Encode(object)
	if err != nil {
		return "", err
	}

	sb := strings.Builder{}

	sb.WriteString(fmt.Sprintf("data: " + buff.String() + "\n\n"))

	return sb.String(), nil
}
