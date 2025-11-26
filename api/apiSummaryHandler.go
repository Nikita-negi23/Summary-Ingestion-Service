package api

import (
	"fmt"
	"net/http"
)

// this function handles the pdf upload request
func SummaryUploadHandler(w http.ResponseWriter, r *http.Request) {
	//1) Validate the request and save the file in project directory folder.
	filePath := ValidateRequestAndSave(w, r)

	//2) Get the raw data and summarized data from the uploaded PDF (integrating AI)
	text, summary := GetPDFSummary(w, r, filePath)

	fmt.Println("Raw PDF Data:", text)
	fmt.Println("Summary PDF Data:", summary)

	//3) Store the raw data and summarized data in the database.
	tenantName := r.FormValue("tenant_name")
	// check this tenant name is present in Master Database or not, if not present then create database for this tenant in mongoDB and insert
	// record in Master Database.
	UpsertTenantData(tenantName, text, summary, filePath)
}
