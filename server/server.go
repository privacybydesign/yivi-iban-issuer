package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/google/uuid"
)

const ErrorPhoneNumberFormat = "error:phone-number-format"
const ErrorRateLimit = "error:ratelimit"
const ErrorCannotValidateToken = "error:cannot-validate-token"
const ErrorAddressMalformed = "error:address-malformed"
const ErrorInternal = "error:internal"
const ErrorSendingSms = "error:sending-sms"

type ServerConfig struct {
	Host           string `json:"host"`
	Port           int    `json:"port"`
	UseTls         bool   `json:"use_tls,omitempty"`
	TlsPrivKeyPath string `json:"tls_priv_key_path,omitempty"`
	TlsCertPath    string `json:"tls_cert_path,omitempty"`
}

type ServerState struct {
	ibanChecker      IbanChecker
	jwtCreator       JwtCreator
	transactionCache map[string]string
}

type Server struct {
	server *http.Server
	config ServerConfig
}

func (s *Server) ListenAndServe() error {
	if s.config.UseTls {
		return s.server.ListenAndServeTLS(s.config.TlsCertPath, s.config.TlsPrivKeyPath)
	} else {
		return s.server.ListenAndServe()
	}
}

func (s *Server) Stop() error {
	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	return s.server.Shutdown(ctx)
}

func NewServer(state *ServerState, config ServerConfig) (*Server, error) {
	// static file server for the web part on the root
	fs := http.FileServer(http.Dir("../react-cra/build"))

	mux := http.NewServeMux()

	mux.Handle("/", fs)

	// api to handle validating the phone number
	mux.HandleFunc("/ibancheck", func(w http.ResponseWriter, r *http.Request) {
		handleIBANCheck(state, w, r)
	})
	mux.HandleFunc("/status", func(w http.ResponseWriter, r *http.Request) {
		handleGetIBANStatus(state, w, r)
	})

	addr := fmt.Sprintf("%v:%v", config.Host, config.Port)
	server := &http.Server{
		Addr:    addr,
		Handler: mux,
	}

	return &Server{
		server: server,
		config: config,
	}, nil
}

type IBANCheckResponseMessage struct {
	TransactionID           string `json:"transaction_id"`
	IssuerAuthenticationURL string `json:"issuer_authentication_url"`
}

func handleIBANCheck(state *ServerState, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Generate new guid
	entranceCode := uuid.New().String()

	ibanTransaction, err := state.ibanChecker.StartIbanCheck(entranceCode)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to start iban check", err)
		return
	}

	// Add to transaction cache
	fmt.Println("Adding to transaction cache:", ibanTransaction.TransactionID, ibanTransaction.MerchantReference)
	state.transactionCache[ibanTransaction.TransactionID] = ibanTransaction.MerchantReference

	responseMessage := IBANCheckResponseMessage{
		TransactionID:           ibanTransaction.TransactionID,
		IssuerAuthenticationURL: ibanTransaction.IssuerAuthenticationURL,
	}

	payload, err := json.Marshal(responseMessage)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to marshal response message", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

type IBANStatusResponseMessage struct {
	TransactionStatus TransactionStatus `json:"transaction_status"`
	Jwt               string            `json:"jwt"`
}

func handleGetIBANStatus(state *ServerState, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

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
		respondWithErr(w, http.StatusBadRequest, ErrorInternal, "failed to parse json for body of the request", err)
		return
	}

	merchantRef, found := state.transactionCache[input.TransactionID]
	if !found {
		respondWithErr(w, http.StatusBadRequest, ErrorInternal, "transaction not found", err)
		return
	}

	transactionStatus, err := state.ibanChecker.GetStatus(merchantRef, input.TransactionID)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to get iban status", err)
		return
	}

	if transactionStatus == nil {
		respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "transaction status is nil", err)
		return
	}

	IBANStatusResponseMessage := IBANStatusResponseMessage{
		TransactionStatus: *transactionStatus,
	}

	if transactionStatus.Status == "success" {
		// Create JWT
		IBANStatusResponseMessage.Jwt, err = state.jwtCreator.CreateJwt(transactionStatus.Name, transactionStatus.IBAN, transactionStatus.IssuerID)
		if err != nil {
			respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to create jwt", err)
			return
		}
		// Remove from transaction cache
		delete(state.transactionCache, input.TransactionID)
	}

	payload, err := json.Marshal(IBANStatusResponseMessage)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to marshal response message", err)
		return
	}

	w.WriteHeader(http.StatusOK)
	w.Header().Set("Content-Type", "application/json")
	w.Write(payload)
}

func respondWithErr(w http.ResponseWriter, code int, responseBody string, logMsg string, e error) {
	m := fmt.Sprintf("%v: %v", logMsg, e)
	fmt.Println("%s\n -> returning statuscode %d with message %v", m, code, responseBody)
	w.WriteHeader(code)
	if _, err := w.Write([]byte(responseBody)); err != nil {
		fmt.Println("failed to write body to http response: %v", err)
	}
}
