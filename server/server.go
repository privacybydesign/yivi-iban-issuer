package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"

	"github.com/google/uuid"
	"github.com/joho/godotenv"
)

var cmUrl string = ""
var merchantToken string = ""
var returnUrl string = ""
var transactionCache = make(map[string]string)

func main() {
	err := godotenv.Load()
	if err != nil {
		log.Fatal("Error loading .env file")
	}

	cmUrl = os.Getenv("CM_URL")
	if cmUrl == "" {
		log.Fatal("CM_URL is not set in .env file")
		return
	}

	merchantToken = os.Getenv("MERCHANT_TOKEN")
	if merchantToken == "" {
		log.Fatal("MERCHANT_TOKEN is not set in .env file")
		return
	}

	returnUrl = os.Getenv("RETURN_URL")
	if returnUrl == "" {
		log.Fatal("RETURN_URL is not set in .env file")
		return
	}

	http.Handle("/", http.FileServer(http.Dir(os.Getenv("STATIC_DIR"))))
	http.HandleFunc("/ibancheck", handleIBANCheck)
	http.HandleFunc("/status", handleGetIBANStatus)
	http.HandleFunc("/session", handleSession)

	log.Println("server running at 0.0.0.0:4242")
	http.ListenAndServe("0.0.0.0:4242", nil)
}

type IbanCheck struct {
	MerchantToken     string `json:"merchant_token"`
	EntranceCode      string `json:"entrance_code"`
	MerchantReturnUrl string `json:"merchant_return_url"`
}

type IdealTransaction struct {
	TransactionID           string `json:"transaction_id"`
	EntranceCode            string `json:"entrance_code"`
	MerchantReference       string `json:"merchant_reference"`
	IssuerAuthenticationURL string `json:"issuer_authentication_url"`
}

type MerchantTransaction struct {
	MerchantToken     string `json:"merchant_token"`
	TransactionID     string `json:"transaction_id"`
	MerchantReference string `json:"merchant_reference"`
}

type TransactionStatus struct {
	TransactionID string `json:"transaction_id"`
	Status        string `json:"status"`
	IssuerID      string `json:"issuer_id"`
	Name          string `json:"name"`
	IBAN          string `json:"iban"`
}

type IBANCheckResponseMessage struct {
	TransactionID           string `json:"transaction_id"`
	IssuerAuthenticationURL string `json:"issuer_authentication_url"`
}

type ErrorResponseMessage struct {
	Message string `json:"message"`
}

type ErrorResponse struct {
	Error *ErrorResponseMessage `json:"error"`
}

func handleGetIBANStatus(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Define a struct to read the incoming JSON
	var input struct {
		TransactionID string `json:"transaction_id"`
	}

	// Decode the JSON body
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		writeJSONErrorMessage(w, "Invalid JSON", http.StatusBadRequest)
		return
	}

	fmt.Println("Getting status for transaction ID:", input.TransactionID)
	merchantRef, found := transactionCache[input.TransactionID]
	if !found {
		writeJSONErrorMessage(w, "TransactionID not found", http.StatusNotFound)
		return
	}

	merchantTransaction := MerchantTransaction{
		MerchantToken:     merchantToken,
		TransactionID:     input.TransactionID,
		MerchantReference: merchantRef,
	}

	jsonData, err := json.Marshal(merchantTransaction)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Do a request to CM backend.
	fmt.Println("Calling CM with URL:", cmUrl+"transaction")
	bytes, err := callCM("POST", cmUrl+"status", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error calling CM:", err)
		writeJSONErrorMessage(w, "Error calling CM", http.StatusInternalServerError)
		return
	}

	var transactionStatus TransactionStatus
	err = json.Unmarshal(bytes, &transactionStatus)
	if err != nil {
		fmt.Println("Error unmarshaling response:", err)
		return
	}

	fmt.Println("Response:", transactionStatus)
	writeJSON(w, transactionStatus)
}

func handleIBANCheck(w http.ResponseWriter, r *http.Request) {
	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Generate new guid
	guid := uuid.New().String()

	// Create request body
	ibanCheck := IbanCheck{
		MerchantToken:     merchantToken,
		EntranceCode:      guid,
		MerchantReturnUrl: returnUrl,
	}

	fmt.Println(returnUrl)

	jsonData, err := json.Marshal(ibanCheck)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return
	}

	// Do a request to CM backend.
	fmt.Println("Calling CM with URL:", cmUrl+"transaction")
	bytes, err := callCM("POST", cmUrl+"transaction", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error calling CM:", err)
		writeJSONErrorMessage(w, "Error calling CM", http.StatusInternalServerError)
		return
	}

	var ibanTransaction IdealTransaction
	err = json.Unmarshal(bytes, &ibanTransaction)
	if err != nil {
		fmt.Println("Error unmarshaling response:", err)
		return
	}

	// Add to transaction cache
	fmt.Println("Adding to transaction cache:", ibanTransaction.TransactionID, ibanTransaction.MerchantReference)
	transactionCache[ibanTransaction.TransactionID] = ibanTransaction.MerchantReference

	responseMessage := IBANCheckResponseMessage{
		TransactionID:           ibanTransaction.TransactionID,
		IssuerAuthenticationURL: ibanTransaction.IssuerAuthenticationURL,
	}
	fmt.Println("Response:", responseMessage)
	writeJSON(w, responseMessage)
}

func callCM(method string, url string, body io.Reader) ([]byte, error) {
	// Create the request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Create the HTTP client and execute the request
	client := &http.Client{}
	resp, err := client.Do(req)
	if err != nil {
		fmt.Println("Error making request:", err)
		return nil, err
	}
	defer resp.Body.Close()

	// Read the response body
	bytes, err := io.ReadAll(resp.Body)
	if err != nil {
		fmt.Println("Error reading response:", err)
		return nil, err
	}

	// Print status and response
	fmt.Println("Status:", resp.Status)
	fmt.Println("Response:", string(bytes))
	return bytes, nil
}

func writeJSON(w http.ResponseWriter, v interface{}) {
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(v); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		log.Printf("json.NewEncoder.Encode: %v", err)
		return
	}
	w.Header().Set("Content-Type", "application/json")
	if _, err := io.Copy(w, &buf); err != nil {
		log.Printf("io.Copy: %v", err)
		return
	}
}

func writeJSONError(w http.ResponseWriter, v interface{}, code int) {
	w.WriteHeader(code)
	writeJSON(w, v)
}

func writeJSONErrorMessage(w http.ResponseWriter, message string, code int) {
	resp := &ErrorResponse{
		Error: &ErrorResponseMessage{
			Message: message,
		},
	}
	writeJSONError(w, resp, code)
}
