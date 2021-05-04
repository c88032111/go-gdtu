// Copyright 2016 The go-gdtu Authors
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
	"io/ioutil"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/cespare/cp"
)

// These tests are 'smoke tests' for the account related
// subcommands and flags.
//
// For most tests, the test files from package accounts
// are copied into a temporary keystore directory.

func tmpDatadirWithKeystore(t *testing.T) string {
	datadir := tmpdir(t)
	keystore := filepath.Join(datadir, "keystore")
	source := filepath.Join("..", "..", "accounts", "keystore", "testdata", "keystore")
	if err := cp.CopyAll(keystore, source); err != nil {
		t.Fatal(err)
	}
	return datadir
}

func TestAccountListEmpty(t *testing.T) {
	ggdtu := runGgdtu(t, "account", "list")
	ggdtu.ExpectExit()
}

func TestAccountList(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	ggdtu := runGgdtu(t, "account", "list", "--datadir", datadir)
	defer ggdtu.ExpectExit()
	if runtime.GOOS == "windows" {
		ggdtu.Expect(`
Account #0: {7ef5a6135f1fd6a02593eedc869c6d41d934aef8} keystore://{{.Datadir}}\keystore\UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8
Account #1: {f466859ead1932d743d622cb74fc058882e8648a} keystore://{{.Datadir}}\keystore\aaa
Account #2: {289d485d9771714cce91d3393d764e1311907acc} keystore://{{.Datadir}}\keystore\zzz
`)
	} else {
		ggdtu.Expect(`
Account #0: {7ef5a6135f1fd6a02593eedc869c6d41d934aef8} keystore://{{.Datadir}}/keystore/UTC--2016-03-22T12-57-55.920751759Z--7ef5a6135f1fd6a02593eedc869c6d41d934aef8
Account #1: {f466859ead1932d743d622cb74fc058882e8648a} keystore://{{.Datadir}}/keystore/aaa
Account #2: {289d485d9771714cce91d3393d764e1311907acc} keystore://{{.Datadir}}/keystore/zzz
`)
	}
}

func TestAccountNew(t *testing.T) {
	ggdtu := runGgdtu(t, "account", "new", "--lightkdf")
	defer ggdtu.ExpectExit()
	ggdtu.Expect(`
Your new account is locked with a password. Please give a password. Do not forget this password.
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
Repeat password: {{.InputLine "foobar"}}

Your new key was generated
`)
	ggdtu.ExpectRegexp(`
Public address of the key:   0x[0-9a-fA-F]{40}
Path of the secret key file: .*UTC--.+--[0-9a-f]{40}

- You can share your public address with anyone. Others need it to interact with you.
- You must NEVER share the secret key with anyone! The key controls access to your funds!
- You must BACKUP your key file! Without the key, it's impossible to access account funds!
- You must REMEMBER your password! Without the password, it's impossible to decrypt the key!
`)
}

func TestAccountImport(t *testing.T) {
	tests := []struct{ name, key, output string }{
		{
			name:   "correct account",
			key:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef",
			output: "Address: {fcad0b19bb29d4674531d6f115237e16afce377c}\n",
		},
		{
			name:   "invalid character",
			key:    "0123456789abcdef0123456789abcdef0123456789abcdef0123456789abcdef1",
			output: "Fatal: Failed to load the private key: invalid character '1' at end of key file\n",
		},
	}
	for _, test := range tests {
		test := test
		t.Run(test.name, func(t *testing.T) {
			t.Parallel()
			importAccountWithExpect(t, test.key, test.output)
		})
	}
}

func importAccountWithExpect(t *testing.T, key string, expected string) {
	dir := tmpdir(t)
	keyfile := filepath.Join(dir, "key.prv")
	if err := ioutil.WriteFile(keyfile, []byte(key), 0600); err != nil {
		t.Error(err)
	}
	passwordFile := filepath.Join(dir, "password.txt")
	if err := ioutil.WriteFile(passwordFile, []byte("foobar"), 0600); err != nil {
		t.Error(err)
	}
	ggdtu := runGgdtu(t, "account", "import", keyfile, "-password", passwordFile)
	defer ggdtu.ExpectExit()
	ggdtu.Expect(expected)
}

func TestAccountNewBadRepeat(t *testing.T) {
	ggdtu := runGgdtu(t, "account", "new", "--lightkdf")
	defer ggdtu.ExpectExit()
	ggdtu.Expect(`
Your new account is locked with a password. Please give a password. Do not forget this password.
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "somgdtuing"}}
Repeat password: {{.InputLine "somgdtuing else"}}
Fatal: Passwords do not match
`)
}

func TestAccountUpdate(t *testing.T) {
	datadir := tmpDatadirWithKeystore(t)
	ggdtu := runGgdtu(t, "account", "update",
		"--datadir", datadir, "--lightkdf",
		"f466859ead1932d743d622cb74fc058882e8648a")
	defer ggdtu.ExpectExit()
	ggdtu.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
Please give a new password. Do not forget this password.
Password: {{.InputLine "foobar2"}}
Repeat password: {{.InputLine "foobar2"}}
`)
}

func TestWalletImport(t *testing.T) {
	ggdtu := runGgdtu(t, "wallet", "import", "--lightkdf", "testdata/guswallet.json")
	defer ggdtu.ExpectExit()
	ggdtu.Expect(`
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foo"}}
Address: {d4584b5f6229b7be90727b0fc8c6b91bb427821f}
`)

	files, err := ioutil.ReadDir(filepath.Join(ggdtu.Datadir, "keystore"))
	if len(files) != 1 {
		t.Errorf("expected one key file in keystore directory, found %d files (error: %v)", len(files), err)
	}
}

func TestWalletImportBadPassword(t *testing.T) {
	ggdtu := runGgdtu(t, "wallet", "import", "--lightkdf", "testdata/guswallet.json")
	defer ggdtu.ExpectExit()
	ggdtu.Expect(`
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "wrgdtu"}}
Fatal: could not decrypt key with given password
`)
}

func TestUnlockFlag(t *testing.T) {
	ggdtu := runMinimalGgdtu(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a", "js", "testdata/empty.js")
	ggdtu.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
`)
	ggdtu.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0xf466859eAD1932D743d622CB74FC058882E8648A",
	}
	for _, m := range wantMessages {
		if !strings.Contains(ggdtu.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagWrgdtuPassword(t *testing.T) {
	ggdtu := runMinimalGgdtu(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a", "js", "testdata/empty.js")

	defer ggdtu.ExpectExit()
	ggdtu.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "wrgdtu1"}}
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 2/3
Password: {{.InputLine "wrgdtu2"}}
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 3/3
Password: {{.InputLine "wrgdtu3"}}
Fatal: Failed to unlock account f466859ead1932d743d622cb74fc058882e8648a (could not decrypt key with given password)
`)
}

// https://github.com/c88032111/go-gdtu/issues/1785
func TestUnlockFlagMultiIndex(t *testing.T) {
	ggdtu := runMinimalGgdtu(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a", "--unlock", "0,2", "js", "testdata/empty.js")

	ggdtu.Expect(`
Unlocking account 0 | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
Unlocking account 2 | Attempt 1/3
Password: {{.InputLine "foobar"}}
`)
	ggdtu.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x7EF5A6135f1FD6a02593eEdC869c6D41D934aef8",
		"=0x289d485D9771714CCe91D3393D764E1311907ACc",
	}
	for _, m := range wantMessages {
		if !strings.Contains(ggdtu.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagPasswordFile(t *testing.T) {
	ggdtu := runMinimalGgdtu(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a", "--password", "testdata/passwords.txt", "--unlock", "0,2", "js", "testdata/empty.js")

	ggdtu.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0x7EF5A6135f1FD6a02593eEdC869c6D41D934aef8",
		"=0x289d485D9771714CCe91D3393D764E1311907ACc",
	}
	for _, m := range wantMessages {
		if !strings.Contains(ggdtu.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagPasswordFileWrgdtuPassword(t *testing.T) {
	ggdtu := runMinimalGgdtu(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a", "--password",
		"testdata/wrgdtu-passwords.txt", "--unlock", "0,2")
	defer ggdtu.ExpectExit()
	ggdtu.Expect(`
Fatal: Failed to unlock account 0 (could not decrypt key with given password)
`)
}

func TestUnlockFlagAmbiguous(t *testing.T) {
	store := filepath.Join("..", "..", "accounts", "keystore", "testdata", "dupes")
	ggdtu := runMinimalGgdtu(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a", "--keystore",
		store, "--unlock", "f466859ead1932d743d622cb74fc058882e8648a",
		"js", "testdata/empty.js")
	defer ggdtu.ExpectExit()

	// Helper for the expect template, returns absolute keystore path.
	ggdtu.SetTemplateFunc("keypath", func(file string) string {
		abs, _ := filepath.Abs(filepath.Join(store, file))
		return abs
	})
	ggdtu.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "foobar"}}
Multiple key files exist for address f466859ead1932d743d622cb74fc058882e8648a:
   keystore://{{keypath "1"}}
   keystore://{{keypath "2"}}
Testing your password against all of them...
Your password unlocked keystore://{{keypath "1"}}
In order to avoid this warning, you need to remove the following duplicate key files:
   keystore://{{keypath "2"}}
`)
	ggdtu.ExpectExit()

	wantMessages := []string{
		"Unlocked account",
		"=0xf466859eAD1932D743d622CB74FC058882E8648A",
	}
	for _, m := range wantMessages {
		if !strings.Contains(ggdtu.StderrText(), m) {
			t.Errorf("stderr text does not contain %q", m)
		}
	}
}

func TestUnlockFlagAmbiguousWrgdtuPassword(t *testing.T) {
	store := filepath.Join("..", "..", "accounts", "keystore", "testdata", "dupes")
	ggdtu := runMinimalGgdtu(t, "--port", "0", "--ipcdisable", "--datadir", tmpDatadirWithKeystore(t),
		"--unlock", "f466859ead1932d743d622cb74fc058882e8648a", "--keystore",
		store, "--unlock", "f466859ead1932d743d622cb74fc058882e8648a")

	defer ggdtu.ExpectExit()

	// Helper for the expect template, returns absolute keystore path.
	ggdtu.SetTemplateFunc("keypath", func(file string) string {
		abs, _ := filepath.Abs(filepath.Join(store, file))
		return abs
	})
	ggdtu.Expect(`
Unlocking account f466859ead1932d743d622cb74fc058882e8648a | Attempt 1/3
!! Unsupported terminal, password will be echoed.
Password: {{.InputLine "wrgdtu"}}
Multiple key files exist for address f466859ead1932d743d622cb74fc058882e8648a:
   keystore://{{keypath "1"}}
   keystore://{{keypath "2"}}
Testing your password against all of them...
Fatal: None of the listed files could be unlocked.
`)
	ggdtu.ExpectExit()
}
