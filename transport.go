package main

import (
	"encoding/json"
	"fmt"
	"net/http"

	"github.com/geometryzen/stemcstudio-arXiv-sdk-go"
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
	// Maybe there is a more efficient way to be a gateway?
	return func(w http.ResponseWriter, r *http.Request) {
		var payload arXiv.Submission
		decoder := json.NewDecoder(r.Body)
		err := decoder.Decode(&payload)
		if err != nil {
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		_, err = service.Submit(&payload)
		if err != nil {
			fmt.Println("err  : ", err)
			http.Error(w, "Bad request", http.StatusBadRequest)
			return
		}
		http.Error(w, "OK", http.StatusOK)
	}
}
