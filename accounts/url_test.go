// Copyright 2018 The go-gdtu Authors
// This file is part of the go-gdtu library.
//
// The go-gdtu library is free software: you can redistribute it and/or modify
// it under the terms of the GNU Lesser General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// The go-gdtu library is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU Lesser General Public License for more details.
//
// You should have received a copy of the GNU Lesser General Public License
// algdtu with the go-gdtu library. If not, see <http://www.gnu.org/licenses/>.

package accounts

import (
	"testing"
)

func TestURLParsing(t *testing.T) {
	url, err := parseURL("https://gdtu2020.com")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "gdtu2020.com" {
		t.Errorf("expected: %v, got: %v", "gdtu2020.com", url.Path)
	}

	_, err = parseURL("gdtu2020.com")
	if err == nil {
		t.Error("expected err, got: nil")
	}
}

func TestURLString(t *testing.T) {
	url := URL{Scheme: "https", Path: "gdtu2020.com"}
	if url.String() != "https://gdtu2020.com" {
		t.Errorf("expected: %v, got: %v", "https://gdtu2020.com", url.String())
	}

	url = URL{Scheme: "", Path: "gdtu2020.com"}
	if url.String() != "gdtu2020.com" {
		t.Errorf("expected: %v, got: %v", "gdtu2020.com", url.String())
	}
}

func TestURLMarshalJSON(t *testing.T) {
	url := URL{Scheme: "https", Path: "gdtu2020.com"}
	json, err := url.MarshalJSON()
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if string(json) != "\"https://gdtu2020.com\"" {
		t.Errorf("expected: %v, got: %v", "\"https://gdtu2020.com\"", string(json))
	}
}

func TestURLUnmarshalJSON(t *testing.T) {
	url := &URL{}
	err := url.UnmarshalJSON([]byte("\"https://gdtu2020.com\""))
	if err != nil {
		t.Errorf("unexpcted error: %v", err)
	}
	if url.Scheme != "https" {
		t.Errorf("expected: %v, got: %v", "https", url.Scheme)
	}
	if url.Path != "gdtu2020.com" {
		t.Errorf("expected: %v, got: %v", "https", url.Path)
	}
}

func TestURLComparison(t *testing.T) {
	tests := []struct {
		urlA   URL
		urlB   URL
		expect int
	}{
		{URL{"https", "gdtu2020.com"}, URL{"https", "gdtu2020.com"}, 0},
		{URL{"http", "gdtu2020.com"}, URL{"https", "gdtu2020.com"}, -1},
		{URL{"https", "gdtu2020.com/a"}, URL{"https", "gdtu2020.com"}, 1},
		{URL{"https", "abc.org"}, URL{"https", "gdtu2020.com"}, -1},
	}

	for i, tt := range tests {
		result := tt.urlA.Cmp(tt.urlB)
		if result != tt.expect {
			t.Errorf("test %d: cmp mismatch: expected: %d, got: %d", i, tt.expect, result)
		}
	}
}
