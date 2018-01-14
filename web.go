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
	"os"
	"strings"

	"github.com/aws/aws-sdk-go/aws"

	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/cloudsearchdomain"

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

type searchRequest struct {
	Query string `json:"query"`
}

type searchRef struct {
	HRef     string   `json:"href"`
	Owner    string   `json:"owner"`
	GistID   string   `json:"gistId"`
	Title    string   `json:"title"`
	Author   string   `json:"author"`
	Keywords []string `json:"keywords"`
}

type searchResponse struct {
	Found int64       `json:"found"`
	Start int64       `json:"start"`
	Refs  []searchRef `json:"refs"`
}

type submissionPayload struct {
	Author   string   `json:"author"`
	GistID   string   `json:"gistId"`
	Keywords []string `json:"keywords"`
	Owner    string   `json:"owner"`
	Title    string   `json:"title"`
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

func mapToString(fields map[string][]*string, name string) string {
	xs := fields[name]
	if len(xs) > 0 {
		return aws.StringValue(xs[0])
	}
	return ""
}

func mapToStrings(fields map[string][]*string, name string) []string {
	return aws.StringValueSlice(fields[name])
}

/**
 * Handle POST "/search" with body {query: "..."}
 */
func makeSearch(sess *session.Session) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload searchRequest
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&payload)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}

		csd := cloudsearchdomain.New(sess, &aws.Config{Endpoint: aws.String("search-doodle-ref-xieragrgc2gcnrcog3r6bme75u.us-east-1.cloudsearch.amazonaws.com")})

		size := int64(30)
		data, err := csd.Search(&cloudsearchdomain.SearchInput{Query: aws.String(payload.Query), Size: &size})
		if err != nil {
			fmt.Println("err  : ", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		response := searchResponse{Found: *data.Hits.Found, Start: *data.Hits.Start}
		for _, record := range data.Hits.Hit {
			href := *record.Id
			owner := mapToString(record.Fields, "ownerkey")
			gistID := mapToString(record.Fields, "resourcekey")
			title := mapToString(record.Fields, "title")
			author := mapToString(record.Fields, "author")
			keywords := aws.StringValueSlice(record.Fields["keywords"])
			response.Refs = append(response.Refs, searchRef{Author: author, GistID: gistID, HRef: href, Owner: owner, Title: title, Keywords: keywords})
		}
		json.NewEncoder(w).Encode(&response)
	}
}

/**
 * Handle POST "/submissions" with body {author, credentials, gistId, keywords, owner, title}
 */
func submissions(w http.ResponseWriter, r *http.Request) {
	var payload submissionPayload
	decoder := json.NewDecoder(r.Body)
	err := decoder.Decode(&payload)
	if err != nil {
		http.Error(w, "Bad request", http.StatusBadRequest)
		return
	}
	// The search string is now in payload.Query.
	fmt.Println("POST /submissions")
	fmt.Println("Author  : ", payload.Author)
	fmt.Println("GistID  : ", payload.GistID)
	fmt.Println("Keywords: ", payload.Keywords)
	fmt.Println("Owner   : ", payload.Owner)
	fmt.Println("Title   : ", payload.Title)

	http.Error(w, "Bad request", http.StatusBadRequest)
}

func main() {
	flag.Parse()

	router := mux.NewRouter()

	// GitHub configuration.
	githubAccessKey := os.Getenv("GITHUB_APPLICATION_CLIENT_ID")
	githubSecretKey := os.Getenv("GITHUB_APPLICATION_CLIENT_SECRET")
	fmt.Printf("GITHUB_APPLICATION_CLIENT_ID => %s\n", githubAccessKey)
	fmt.Printf("len(GITHUB_APPLICATION_CLIENT_SECRET) => %d\n", len(githubSecretKey))

	awsAccessKey := os.Getenv("AWS_ACCESS_KEY_ID")
	awsSecretKey := os.Getenv("AWS_SECRET_ACCESS_KEY")
	fmt.Printf("AWS_ACCESS_KEY_ID => %s\n", awsAccessKey)
	fmt.Printf("len(AWS_SECRET_ACCESS_KEY) => %d\n", len(awsSecretKey))

	// AWS configuration
	sess, _ := session.NewSession(&aws.Config{Region: aws.String("us-east-1")})

	router.HandleFunc("/github_callback", githubCallback)

	// We want to handle /authenticate/code
	router.HandleFunc("/authenticate/{code}", makeExchange(githubAccessKey, githubSecretKey))

	router.HandleFunc("/search", makeSearch(sess))
	router.HandleFunc("/submissions", submissions)

	router.PathPrefix("/").Handler(withCookies(http.FileServer(http.Dir("./generated")), githubAccessKey))

	server := &http.Server{
		Addr:    "0.0.0.0:8080",
		Handler: router,
	}

	server.ListenAndServe()
}
