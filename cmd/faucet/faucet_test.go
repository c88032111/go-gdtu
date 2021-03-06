// Copyright 2021 The go-gdtu Authors
// This file is part of go-gdtu.
//
// go-gdtu is free software: you can redistribute it and/or modify
// it under the terms of the GNU General Public License as published by
// the Free Software Foundation, either version 3 of the License, or
// (at your option) any later version.
//
// go-gdtu is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE. See the
// GNU General Public License for more details.
//
// You should have received a copy of the GNU General Public License
// algdtu with go-gdtu. If not, see <http://www.gnu.org/licenses/>.

package main

import (
	"testing"

	"github.com/c88032111/go-gdtu/common"
)

func TestFacebook(t *testing.T) {
	for _, tt := range []struct {
		url  string
		want common.Address
	}{
		{
			"https://www.facebook.com/fooz.gazonk/posts/2837228539847129",
			common.HexToAddress("gdDeadDeaDDeaDbEefbEeFbEEfBeeFBeefBeeFbEEF"),
		},
	} {
		_, _, gotAddress, err := authFacebook(tt.url)
		if err != nil {
			t.Fatal(err)
		}
		if gotAddress != tt.want {
			t.Fatalf("address wrgdtu, have %v want %v", gotAddress, tt.want)
		}
	}
}
