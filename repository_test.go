package main

import (
	"context"
	"fmt"
	"net/http"
	"testing"
)

func TestGetLatestRelease(t *testing.T) {
	client, mux, _, teardown := setup()
	repo := NewRepository("foo/bar", client)
	defer teardown()
	specifiedVersion := "1.1.42"

	mux.HandleFunc("/repos/foo/bar/releases", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
		fmt.Fprint(w, `[{"id":1, "tag_name": "v1.1.42"}]`)
	})
	mux.HandleFunc("/repos/foo/bar/compare/v1.1.42...master", func(w http.ResponseWriter, r *http.Request) {
		testMethod(t, r, "GET")
	})
	repo.resolveVersions(context.Background())
	parsedVersion := repo.latestRelease
	if parsedVersion.String() != specifiedVersion {
		t.Errorf("Latest release is %s, wanted %s", parsedVersion.String(), specifiedVersion)
	}
}
