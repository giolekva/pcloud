package main

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"strings"
	"unicode"

	"github.com/gorilla/mux"
)

type APIServer struct {
	r          *mux.Router
	serv       *http.Server
	kratosAddr string
}

type ErrorResponse struct {
	Error struct {
		Code    int    `json:"code"`
		Status  string `json:"status"`
		Message string `json:"message"`
	} `json:"error"`
}

func NewAPIServer(port int, kratosAddr string) *APIServer {
	r := mux.NewRouter()
	serv := &http.Server{
		Addr:    fmt.Sprintf(":%d", port),
		Handler: r,
	}
	return &APIServer{r, serv, kratosAddr}
}

func (s *APIServer) Start() error {
	s.r.Path("/identities").Methods(http.MethodPost).HandlerFunc(s.identityCreate)
	return s.serv.ListenAndServe()
}

type kratosIdentityCreateReq struct {
	Credentials struct {
		Password struct {
			Config struct {
				Password string `json:"password"`
			} `json:"config"`
		} `json:"password"`
	} `json:"credentials"`
	SchemaID string `json:"schema_id"`
	State    string `json:"state"`
	Traits   struct {
		Username string `json:"username"`
	} `json:"traits"`
}

type identityCreateReq struct {
	Username string `json:"username,omitempty"`
	Password string `json:"password,omitempty"`
}

func extractKratosErrorMessage(errResp ErrorResponse) []ValidationError {
	var errors []ValidationError
	switch errResp.Error.Status {
	case "Conflict":
		errors = append(errors, ValidationError{"username", "Username is not available."})
	case "Bad Request":
		errors = append(errors, ValidationError{"username", "Username is less than 3 characters."})
	default:
		errors = append(errors, ValidationError{"username", "Unexpexted Error."})
	}
	return errors
}

type ValidationError struct {
	Field   string `json:"field"`
	Message string `json:"message"`
}

type CombinedErrors struct {
	Errors []ValidationError `json:"errors"`
}

func validateUsername(username string) []ValidationError {
	var errors []ValidationError
	if len(username) < 3 {
		errors = append(errors, ValidationError{"username", "Username must be at least 3 characters long."})
	}
	// TODO other validations
	return errors
}

func validatePassword(password string) []ValidationError {
	var errors []ValidationError
	if len(password) < 20 {
		errors = append(errors, ValidationError{"password", "Password must be at least 20 characters long."})
	}
	digit := false
	lowerCase := false
	upperCase := false
	special := false
	for _, c := range password {
		if unicode.IsDigit(c) {
			digit = true
		} else if unicode.IsLower(c) {
			lowerCase = true
		} else if unicode.IsUpper(c) {
			upperCase = true
		} else if strings.Contains(" !\"#$%&'()*+,-./:;<=>?@[\\]^_{|}~", string(c)) {
			special = true
		}
	}
	if !digit || !lowerCase || !upperCase || !special {
		errors = append(errors, ValidationError{"password", "Password must contain at least one ditig, lower/upper and special character"})
	}
	// TODO other validations
	return errors
}

func replyWithErrors(w http.ResponseWriter, errors []ValidationError) {
	response := CombinedErrors{Errors: errors}
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusBadRequest)
	if err := json.NewEncoder(w).Encode(response); err != nil {
		http.Error(w, "failed to decode", http.StatusInternalServerError)
		return
	}
}

func (s *APIServer) identityCreate(w http.ResponseWriter, r *http.Request) {
	var req identityCreateReq
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "request can not be parsed", http.StatusBadRequest)
		return
	}
	usernameErrors := validateUsername(req.Username)
	passwordErrors := validatePassword(req.Password)
	allErrors := append(usernameErrors, passwordErrors...)
	if len(allErrors) > 0 {
		replyWithErrors(w, allErrors)
		return
	}
	var kreq kratosIdentityCreateReq
	kreq.Credentials.Password.Config.Password = req.Password
	kreq.SchemaID = "user"
	kreq.State = "active"
	kreq.Traits.Username = req.Username
	var buf bytes.Buffer
	if err := json.NewEncoder(&buf).Encode(kreq); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}
	resp, err := http.Post(s.identitiesEndpoint(), "application/json", &buf)
	if err != nil {
		http.Error(w, "failed", http.StatusInternalServerError)
		return
	}
	if resp.StatusCode != http.StatusCreated {
		var e ErrorResponse
		if err := json.NewDecoder(resp.Body).Decode(&e); err != nil {
			http.Error(w, "failed to decode", http.StatusInternalServerError)
			return
		}
		errorMessages := extractKratosErrorMessage(e)
		replyWithErrors(w, errorMessages)
		return
	}
}

func (s *APIServer) identitiesEndpoint() string {
	return fmt.Sprintf("%s/admin/identities", s.kratosAddr)
}
