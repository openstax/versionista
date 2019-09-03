package main

import (
	"fmt"
	"testing"
	"net/http"
)

func TestGetLatestRelease(t *testing.T) {
	client, mux, _, teardown := setup()
	repo := NewRepository("foo/bar", client)
	defer teardown()
	specifiedVersion := "1.1.42"

	mux.HandleFunc("/repos/foo/bar/releases/latest", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprintf(w, `{"id":1, "tag_name": "%s"}`, specifiedVersion)
	})
	mux.HandleFunc("/repos/foo/bar/compare/v1.1.42...master",func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
	})
	repo.fetch()
	parsedVersion := repo.latestRelease
	if parsedVersion.String() != specifiedVersion {
		t.Errorf("Latest release is %s, wanted %s", parsedVersion.String(), specifiedVersion)
	}
}
