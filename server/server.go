package main

import (
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"time"

	"github.com/google/uuid"
	"github.com/gorilla/mux"
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
	StaticPath     string `json:"static_path"`
	UseTls         bool   `json:"use_tls,omitempty"`
	TlsPrivKeyPath string `json:"tls_priv_key_path,omitempty"`
	TlsCertPath    string `json:"tls_cert_path,omitempty"`
}

type ServerState struct {
	ibanChecker  IbanChecker
	jwtCreator   JwtCreator
	tokenStorage TokenStorage
}

type spaHandler struct {
	staticPath string
	indexPath  string
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

// ServeHTTP inspects the URL path to locate a file within the static dir
// on the SPA handler. If a file is found, it will be served. If not, the
// file located at the index path on the SPA handler will be served. This
// is suitable behavior for serving an SPA (single page application).
// https://github.com/gorilla/mux?tab=readme-ov-file#serving-single-page-applications
func (h spaHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Join internally call path.Clean to prevent directory traversal
	path := filepath.Join(h.staticPath, r.URL.Path)
	fmt.Println("Serving file:", path)
	// check whether a file exists or is a directory at the given path
	fi, err := os.Stat(path)
	if os.IsNotExist(err) || fi.IsDir() {
		// file does not exist or path is a directory, serve index.html
		http.ServeFile(w, r, filepath.Join(h.staticPath, h.indexPath))
		return
	}

	if err != nil {
		// if we got an error (that wasn't that the file doesn't exist) stating the
		// file, return a 500 internal server error and stop
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// otherwise, use http.FileServer to serve the static file
	http.FileServer(http.Dir(h.staticPath)).ServeHTTP(w, r)
}

func NewServer(state *ServerState, config ServerConfig) (*Server, error) {
	router := mux.NewRouter()

	router.HandleFunc("/api/health", func(w http.ResponseWriter, r *http.Request) {
		json.NewEncoder(w).Encode(map[string]bool{"ok": true})
	})

	router.HandleFunc("/api/ibancheck", func(w http.ResponseWriter, r *http.Request) {
		handleIBANCheck(state, w, r)
	})
	router.HandleFunc("/api/status", func(w http.ResponseWriter, r *http.Request) {
		handleGetIBANStatus(state, w, r)
	})

	spa := spaHandler{staticPath: config.StaticPath, indexPath: "index.html"}
	router.PathPrefix("/").Handler(spa)

	addr := fmt.Sprintf("%v:%v", config.Host, config.Port)
	srv := &http.Server{
		Handler: router,
		Addr:    addr,
		// Good practice: enforce timeouts for servers you create!
		WriteTimeout: 15 * time.Second,
		ReadTimeout:  15 * time.Second,
	}

	return &Server{
		server: srv,
		config: config,
	}, nil
}

type IBANCheckResponseMessage struct {
	TransactionID           TransactonId `json:"transaction_id"`
	IssuerAuthenticationURL string       `json:"issuer_authentication_url"`
}

func handleIBANCheck(state *ServerState, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Generate new guid
	entranceCode := uuid.New().String()

	// Define a struct to read the incoming JSON
	var input struct {
		Language string `json:"language"`
	}

	// Decode the JSON body
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		respondWithErr(w, http.StatusBadRequest, ErrorInternal, "failed to parse json for body of the request", err)
		return
	}

	ibanTransaction, err := state.ibanChecker.StartIbanCheck(entranceCode, input.Language)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to start iban check", err)
		return
	}

	// Add to transaction cache
	fmt.Println("Adding to transaction cache:", ibanTransaction.TransactionID, ibanTransaction.MerchantReference)
	err = state.tokenStorage.StoreToken(ibanTransaction.TransactionID, ibanTransaction.MerchantReference)
	if err != nil {
		respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to store token in cache", err)
		return
	}
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

// handles a POST request to get the status of an IBAN check
// Expects a JSON body with a "transaction_id" field
// Returns a JSON response with the transaction status and JWT if successful
func handleGetIBANStatus(state *ServerState, w http.ResponseWriter, r *http.Request) {
	defer r.Body.Close()

	if r.Method != "POST" {
		http.Error(w, http.StatusText(http.StatusMethodNotAllowed), http.StatusMethodNotAllowed)
		return
	}

	// Define a struct to read the incoming JSON
	var input struct {
		TransactionID TransactonId `json:"transaction_id"`
	}

	// Decode the JSON body
	err := json.NewDecoder(r.Body).Decode(&input)
	if err != nil {
		respondWithErr(w, http.StatusBadRequest, ErrorInternal, "failed to parse json for body of the request", err)
		return
	}

	merchantRef, err := state.tokenStorage.RetrieveToken(input.TransactionID)
	if err != nil {
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
		err = state.tokenStorage.RemoveToken(input.TransactionID)
		if err != nil {
			respondWithErr(w, http.StatusInternalServerError, ErrorInternal, "failed to delete token from cache", err)
			return
		}
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
