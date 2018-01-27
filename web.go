package main

import (
	"bytes"
	"encoding/json"
	"flag"
	"fmt"
	"html/template"
	"io/ioutil"
	"log"
	"net/http"
	"net/url"
	"os"
	"strings"

	"github.com/geometryzen/stemcstudio-arXiv-sdk-go"

	"github.com/gorilla/mux"
)

/**
 * Chain Handler(s) in order to set the cookie required to authenticate with GitHub.
 */
func withCookies(h http.Handler, value string) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		cookie := http.Cookie{
			Name:  "stemcstudio-github-application-client-id",
			Value: value,
		}
		http.SetCookie(w, &cookie)
		h.ServeHTTP(w, r)
	})
}

/**
 * Handler function for the redirect from GitHub.
 */
func githubCallback(w http.ResponseWriter, r *http.Request) {
	files := []string{"templates/github_callback.html"}
	templates := template.Must(template.ParseFiles(files...))
	templates.ExecuteTemplate(w, "githubCallback", "")
}

type exchangePayload struct {
	ClientID     string `json:"client_id"`
	ClientSecret string `json:"client_secret"`
	Code         string `json:"code"`
}

type wrapToken struct {
	Token string `json:"token"`
}

/**
 * Handler function allowing the front-end application to exchange a temporary authorization code for a token.
 * The request comes through the server to avoid exposing the GitHub secret key.
 */
func makeExchange(clientID string, clientSecret string) http.HandlerFunc {
	// We need to return an HTML file that scrapes the code and state on the client application.
	return func(w http.ResponseWriter, r *http.Request) {
		vars := mux.Vars(r)
		code := vars["code"]
		// We now need to make a call to GitHub to exchange the code for a token.
		// TODO: Get all the parameters from the environment.
		data := exchangePayload{ClientID: clientID, ClientSecret: clientSecret, Code: code}
		jsonBytes, err := json.Marshal(data)
		if err != nil {
			log.Fatalf("JSON marshalling failed: %s", err)
		}
		resp, err := http.Post("https://github.com/login/oauth/access_token", "application/json", bytes.NewBuffer(jsonBytes))
		body, _ := ioutil.ReadAll(resp.Body)
		entries := strings.Split(string(body), "&")
		entry := entries[0]
		keyValue := strings.Split(entry, "=")
		token := keyValue[1]
		json.NewEncoder(w).Encode(&wrapToken{Token: token})
		// { "token": token } : { "error": "bad_code" }
	}
}

func main() {
	flag.Parse()

	router := mux.NewRouter()

	// GitHub configuration.
	githubAccessKey := os.Getenv("GITHUB_APPLICATION_CLIENT_ID")
	githubSecretKey := os.Getenv("GITHUB_APPLICATION_CLIENT_SECRET")
	fmt.Printf("GITHUB_APPLICATION_CLIENT_ID => %s\n", githubAccessKey)
	fmt.Printf("len(GITHUB_APPLICATION_CLIENT_SECRET) => %d\n", len(githubSecretKey))

	searchService := arXiv.NewClient(nil)
	baseURL, err := url.Parse("http://localhost:8081")
	if err != nil {
		log.Fatalf("JSON marshalling failed: %s", err)
	}
	searchService.BaseURL = baseURL

	router.HandleFunc("/github_callback", githubCallback)

	// We want to handle /authenticate/code
	router.HandleFunc("/authenticate/{code}", makeExchange(githubAccessKey, githubSecretKey))

	router.HandleFunc("/search", makeSearchHandlerFunc(searchService))
	router.HandleFunc("/submissions", makeSubmitHandlerFunc(searchService))

	router.PathPrefix("/").Handler(withCookies(http.FileServer(http.Dir("./generated")), githubAccessKey))

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}

	server.ListenAndServe()
}
