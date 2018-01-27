package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/geometryzen/stemcstudio-search-sdk-go"
)

func makeSearchHandlerFunc(service arXiv.Service) http.HandlerFunc {
	// Maybe there is a more efficient way to be a gateway?
	return func(w http.ResponseWriter, r *http.Request) {
		var searchRequest arXiv.SearchRequest
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&searchRequest)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		searchResponse, err := service.Search(searchRequest.Query, 30)
		if err != nil {
			fmt.Println("err  : ", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		json.NewEncoder(w).Encode(searchResponse)
	}
}

func makeSubmitHandlerFunc(service arXiv.Service) http.HandlerFunc {
	return func(w http.ResponseWriter, r *http.Request) {
		var payload arXiv.Submission
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&payload)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		packet := arXiv.Submission{Author: payload.Author, GistID: payload.GistID, Keywords: payload.Keywords, Owner: payload.Owner, Title: payload.Title}
		_, err = service.Submit(&packet)
		if err != nil {
			fmt.Println("err  : ", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		http.Error(w, "OK", http.StatusOK)
	}
}
