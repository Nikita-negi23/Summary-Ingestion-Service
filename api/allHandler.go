package api

import (
	"bytes"
	"context"
	"database/sql"
	"encoding/json"
	"fmt"
	"io"
	"io/ioutil"
	"log"
	"net/http"
	"os"
	"strings"
	"time"

	"github.com/ledongthuc/pdf"
	_ "github.com/lib/pq"

	"go.mongodb.org/mongo-driver/bson"
	"go.mongodb.org/mongo-driver/mongo"
	"go.mongodb.org/mongo-driver/mongo/options"
)

func ValidateRequestAndSave(w http.ResponseWriter, r *http.Request) string {
	// Allow only POST method
	if r.Method != http.MethodPost {
		http.Error(w, "Only POST method allowed", http.StatusMethodNotAllowed)
		return ""
	}

	// pdf file size upto 10 MB
	r.ParseMultipartForm(10 << 20)

	// 1️⃣ Get tenant name
	tenantName := r.FormValue("tenant_name")
	if tenantName == "" {
		http.Error(w, "tenant_name is required", http.StatusBadRequest)
		return ""
	}

	// 2️⃣ Get the uploaded file
	file, handler, err := r.FormFile("file")
	if err != nil {
		http.Error(w, "Error retrieving file: "+err.Error(), http.StatusBadRequest)
		return ""
	}
	defer file.Close()

	fmt.Println("Uploaded File:", handler.Filename)
	fmt.Println("Tenant Name:", tenantName)

	// Folder path where PDFs will be stored
	folderPath := "./uploads"

	// Check if folder exists, if not create it
	if _, err := os.Stat(folderPath); os.IsNotExist(err) {
		err = os.MkdirAll(folderPath, os.ModePerm)
		if err != nil {
			http.Error(w, "Failed to create folder: "+err.Error(), http.StatusInternalServerError)
			return ""
		}
	}

	// Create destination file path
	dstPath := fmt.Sprintf("%s/%s", folderPath, handler.Filename)

	// Create file in destination folder
	dst, err := os.Create(dstPath)
	if err != nil {
		http.Error(w, "Failed to create file: "+err.Error(), http.StatusInternalServerError)
		return ""
	}
	defer dst.Close()

	// Copy file content to destination
	_, err = io.Copy(dst, file)
	if err != nil {
		http.Error(w, "Failed to save file: "+err.Error(), http.StatusInternalServerError)
		return ""
	}

	fmt.Fprintf(w, "File uploaded successfully: %s", dstPath)
	return dstPath
}

type summariseRequest struct {
	Inputs     string                 `json:"inputs"`
	Parameters map[string]interface{} `json:"parameters,omitempty"`
}

type summariseResponseItem struct {
	SummaryText string `json:"summary_text,omitempty"`
}

func GetPDFSummary(w http.ResponseWriter, r *http.Request, filePath string) (string, string) {

	//Open and read the PDF content
	f, rpdf, err := pdf.Open(filePath)
	if err != nil {
		http.Error(w, "Error reading PDF: "+err.Error(), http.StatusInternalServerError)
		return "", ""
	}
	defer f.Close()

	fmt.Println("Extracted PDF Text Content:")
	var text string
	for pageIndex := 1; pageIndex <= rpdf.NumPage(); pageIndex++ {
		page := rpdf.Page(pageIndex)
		if page.V.IsNull() {
			continue
		}
		pageText, _ := page.GetPlainText(nil)
		text += pageText + "\n" // append the page content
		fmt.Printf("----- Page %d -----\n%s\n", pageIndex, text)
	}

	//**********************************************************************************************************//
	// Call Hugging Face API for summarization
	apiKey := ""
	model := "facebook/bart-large-cnn"

	url := fmt.Sprintf("https://api-inference.huggingface.co/models/%s", model)

	// Clean and trim the extracted text  -- remove newlines and extra spaces which was causing issues in API call of hugging face.
	cleanText := strings.ReplaceAll(text, "\n", " ")
	cleanText = strings.ReplaceAll(cleanText, "\r", " ")
	cleanText = strings.ReplaceAll(cleanText, "\t", " ")
	cleanText = strings.Join(strings.Fields(cleanText), " ") // remove extra spaces

	// Optional: limit to safe length
	if len(cleanText) > 3500 {
		cleanText = cleanText[:3500]
	}

	// Optional: print for debugging
	fmt.Println("Text sent to Hugging Face (first 500 chars):", cleanText[:500])

	reqBody := summariseRequest{
		Inputs: cleanText, // extracted text from PDF to be summarized
		Parameters: map[string]interface{}{
			"max_length": 150,
			"min_length": 40,
			"do_sample":  false,
		},
	}
	bodyBytes, err := json.Marshal(reqBody)
	if err != nil {
		log.Fatal(err)
	}

	req, err := http.NewRequestWithContext(context.Background(), "POST", url, bytes.NewBuffer(bodyBytes))
	if err != nil {
		log.Fatal(err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		log.Fatal(err)
	}
	defer resp.Body.Close()
	respBytes, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		log.Fatal(err)
	}

	// Check HTTP status code
	if resp.StatusCode != 200 {
		// Try to parse error
		var errResp map[string]interface{}
		if err := json.Unmarshal(respBytes, &errResp); err == nil {
			if msg, ok := errResp["error"]; ok {
				return "", fmt.Sprintf("API Error: %v", msg)
			}
		}
		return "", fmt.Sprintf("API returned status %d", resp.StatusCode)
	}

	// Parse successful response
	var output []summariseResponseItem
	err = json.Unmarshal(respBytes, &output)
	if err != nil {
		log.Fatal("cannot parse response: ", err, " body: ", string(respBytes))
	}

	if len(output) > 0 {
		fmt.Println("Summary Returned :", output[0].SummaryText)
	} else {
		fmt.Println("No summary returned.")
	}

	return cleanText, output[0].SummaryText
}

func UpsertTenantData(tenantName string, text string, summary string, filePath string) {

	// function to connect Postgres sql
	postgres_client := ConnectPostgresSql()
	fmt.Println("Postgres connection")

	// Table and values
	table := "tenant_info"
	checkColumn := "name"
	valueToCheck := tenantName

	// Step 1: SELECT to see if record exists
	var name string
	fmt.Println("Postgres select start")

	err := postgres_client.QueryRow(fmt.Sprintf("SELECT name FROM %s WHERE %s=$1", table, checkColumn), valueToCheck).Scan(&name)

	fmt.Println("Postgres select end", err)

	// function to connect MongoDB
	// mongo_client, ctx := ConnectMongo()
	mongo_client, ctx, cancel, errMongo := ConnectMongo()
	if errMongo != nil {
		log.Fatal("Mongo connection failed:", errMongo)
	}

	defer cancel()                     // cancel the context when done
	defer mongo_client.Disconnect(ctx) // disconnect MongoDB client when done

	fmt.Println("mongo client ")

	var collectionName string
	// create mongodb database and table
	// Dynamically select the database
	db := mongo_client.Database(tenantName)
	fmt.Println("mongodb database done")
	collectionName = "tenant_details"

	if err == sql.ErrNoRows {
		// Step 2: If not exists, INSERT
		_, err := postgres_client.Exec(fmt.Sprintf("INSERT INTO %s (%s) VALUES ($1)", table, checkColumn), valueToCheck)
		fmt.Println("Postgres insert done")

		// Create a new collection (like creating a table)
		err = db.CreateCollection(ctx, collectionName)
		fmt.Println("mongodb collection done")

		if err != nil {
			log.Fatal("Error creating collection:", err)
		}

		fmt.Printf("✅ Database '%s' and collection '%s' created successfully!\n", tenantName, collectionName)
		//////////////////////////////////////////////////////////////////////////////////////////////////////////////////////////

		if err != nil {
			log.Fatal("Error inserting:", err)
		}
		fmt.Println("✅ Record inserted in Master Database for tenant:", valueToCheck)
	} else if err != nil {
		log.Fatal("Error querying:", err)
	} else {
		fmt.Println("Record already exists for tenant:", name)
	}

	collection := db.Collection(collectionName)
	/// insert raw content and summarised content in mongodb
	// Create a document to insert
	doc := bson.M{
		"tenant_name": tenantName,
		"text":        text,
		"summary":     summary,
		"createdAt":   time.Now(),
		"filePath":    filePath,
	}

	// Insert the document
	insertResult, err := collection.InsertOne(ctx, doc)
	if err != nil {
		log.Fatal("Error inserting document:", err)
	}

	fmt.Println("✅ Inserted document with ID:", insertResult.InsertedID)
}

func ConnectMongo() (*mongo.Client, context.Context, context.CancelFunc, error) {
	// MongoDB connection URI
	uri := "mongodb://admin:secret@mongodb:27017/?authSource=admin"

	// Create a context with timeout (you return cancel to call later)
	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)

	// Create a new client
	clientOpts := options.Client().ApplyURI(uri)
	mongoClient, err := mongo.Connect(ctx, clientOpts)
	if err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("error connecting to MongoDB: %v", err)
	}

	// Ping the database to verify connection
	err = mongoClient.Ping(ctx, nil)
	if err != nil {
		cancel()
		return nil, nil, nil, fmt.Errorf("cannot ping MongoDB: %v", err)
	}

	fmt.Println("✅ Successfully connected to MongoDB!")
	return mongoClient, ctx, cancel, nil
}

// Connect go code to Postgres sql.
func ConnectPostgresSql() *sql.DB {
	// Replace with your actual credentials
	host := "postgres" // or container name if Go runs in another container
	port := 5432
	user := "myuser"
	password := "mypassword"
	dbname := "mydb"

	psqlInfo := fmt.Sprintf("host=%s port=%d user=%s password=%s dbname=%s sslmode=disable",
		host, port, user, password, dbname)

	postgres_client, err := sql.Open("postgres", psqlInfo)
	if err != nil {
		log.Fatal("Error opening database:", err)
	}
	// defer postgres_client.Close()

	// Verify connection
	err = postgres_client.Ping()
	if err != nil {
		log.Fatal("Cannot connect to database:", err)
	}

	fmt.Println("✅ Successfully connected to PostgreSQL running in Docker!")
	return postgres_client
}
