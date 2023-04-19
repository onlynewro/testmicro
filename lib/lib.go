package main

import (
	"C"
	"net/http"
	"path/filepath"
	"plugin"
)

type HelloHandler struct{}

func (h *HelloHandler) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	libraryPath, _ := filepath.Abs("../biz/test.so")
	plug, err := plugin.Open(libraryPath)
	if err != nil {
		http.Error(w, "Error loading plugin", http.StatusInternalServerError)
		return
	}

	gethello, err := plug.Lookup("GetHello")
	if err != nil {
		http.Error(w, "Error looking up function", http.StatusInternalServerError)
		return
	}

	gethelloFunc, ok := gethello.(func() string)
	if !ok {
		http.Error(w, "Error asserting function", http.StatusInternalServerError)
		return
	}

	message := gethelloFunc()
	w.Write([]byte(message))
}

var Handler HelloHandler
