package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"
)

type TransactonId string
type MerchantReference string
type MerchantToken string

type IbanCheck struct {
	MerchantToken     MerchantToken `json:"merchant_token"`
	EntranceCode      string        `json:"entrance_code"`
	MerchantReturnUrl string        `json:"merchant_return_url"`
}

type IdealTransaction struct {
	TransactionID           TransactonId      `json:"transaction_id"`
	EntranceCode            string            `json:"entrance_code"`
	MerchantReference       MerchantReference `json:"merchant_reference"`
	IssuerAuthenticationURL string            `json:"issuer_authentication_url"`
}

type MerchantTransaction struct {
	MerchantToken     MerchantToken     `json:"merchant_token"`
	TransactionID     TransactonId      `json:"transaction_id"`
	MerchantReference MerchantReference `json:"merchant_reference"`
}

type TransactionStatus struct {
	TransactionID TransactonId `json:"transaction_id"`
	Status        string       `json:"status"`
	IssuerID      string       `json:"issuer_id"`
	Name          string       `json:"name"`
	IBAN          string       `json:"iban"`
}

type IbanChecker interface {
	GetStatus(merchantRef MerchantReference, transactionId TransactonId) (*TransactionStatus, error)
	StartIbanCheck(entranceCode string, language string) (*IdealTransaction, error)
}

type CmIbanConfig struct {
	BaseUrl       string        `json:"base_url"`
	TimeoutMs     int64         `json:"timeout_ms"`
	ReturnUrl     string        `json:"return_url"`
	MerchantToken MerchantToken `json:"merchant_token"`
}

type CmIbanChecker struct {
	CmIbanConfig
}

func NewCmIbanChecker(config CmIbanConfig) (*CmIbanChecker, error) {
	if !strings.HasPrefix(config.BaseUrl, "https://") {
		return nil, errors.New("CM gateway API endpoint should use https: " + config.BaseUrl)
	}

	return &CmIbanChecker{config}, nil
}

func (s *CmIbanChecker) GetStatus(merchantRef MerchantReference, transactionId TransactonId) (*TransactionStatus, error) {
	merchantTransaction := MerchantTransaction{
		MerchantToken:     s.MerchantToken,
		TransactionID:     transactionId,
		MerchantReference: merchantRef,
	}

	jsonData, err := json.Marshal(merchantTransaction)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return nil, err
	}

	bytes, err := CallCM(s, "POST", s.BaseUrl+"status", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error calling CM:", err)
		return nil, err
	}

	var transactionStatus TransactionStatus
	err = json.Unmarshal(bytes, &transactionStatus)
	if err != nil {
		fmt.Println("Error unmarshaling response:", err)
		return nil, err
	}

	return &transactionStatus, nil
}

func (s *CmIbanChecker) StartIbanCheck(entranceCode string, language string) (*IdealTransaction, error) {
	returnUrl := fmt.Sprintf(s.ReturnUrl, language)
	fmt.Println("Composed returnUrl: ", returnUrl)

	ibanCheck := IbanCheck{
		MerchantToken:     s.MerchantToken,
		EntranceCode:      entranceCode,
		MerchantReturnUrl: returnUrl,
	}

	jsonData, err := json.Marshal(ibanCheck)
	if err != nil {
		fmt.Println("Error marshaling JSON:", err)
		return nil, err
	}

	// Do a request to CM backend.
	fmt.Println("Calling CM with URL:", s.BaseUrl+"transaction")
	bytes, err := CallCM(s, "POST", s.BaseUrl+"transaction", bytes.NewBuffer(jsonData))
	if err != nil {
		fmt.Println("Error calling CM:", err)
		return nil, err
	}

	var ibanTransaction IdealTransaction
	err = json.Unmarshal(bytes, &ibanTransaction)
	if err != nil {
		fmt.Println("Error unmarshaling response:", err)
		return nil, err
	}

	return &ibanTransaction, nil
}

func CallCM(s *CmIbanChecker, method string, url string, body io.Reader) ([]byte, error) {
	// Create the request
	req, err := http.NewRequest(method, url, body)
	if err != nil {
		fmt.Println("Error creating request:", err)
		return nil, err
	}

	// Set headers
	req.Header.Set("Content-Type", "application/json")

	// Create the HTTP client and execute the request
	client := &http.Client{Timeout: time.Duration(s.TimeoutMs) * time.Millisecond}
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
