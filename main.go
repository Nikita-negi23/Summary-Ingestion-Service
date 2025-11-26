package main

import (
	"fmt"
	"net/http"
	api "summary-ingestion-service/api"
)

func main() {
	fmt.Println("Starting Summary Ingestion Service ........")

	http.HandleFunc("/upload", api.SummaryUploadHandler)

	//starting the http server
	fmt.Println("Server starting at port 8080 ..........")
	err := http.ListenAndServe(":8080", nil)
	if err != nil {
		fmt.Println("Error at starting server: ", err)
	}
}
