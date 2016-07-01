// Copyright (c) 2015 SUSE LLC. All rights reserved.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//   http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package backend

import (
	"bytes"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"syscall"
	"testing"
)

func testSlice(t *testing.T, given, expected []string) {
	if len(given) != len(expected) {
		t.Fatalf("Wrong length: given %#v; expected %#v", given, expected)
	}
	for key, value := range given {
		if value != expected[key] {
			t.Fatalf("not the same: given %#v; expected %#v", given, expected)
		}
	}
}

// NOTE: some functions are already covered in other places of this test suite,
// so there's no point to add more tests in this specific file.

func TestCachePath(t *testing.T) {
	home, umask := os.Getenv("HOME"), syscall.Umask(0)
	abs, _ := filepath.Abs(".")

	defer func() {
		syscall.Umask(umask)
		_ = os.RemoveAll(filepath.Join(abs, ".cache"))
		_ = os.Setenv("HOME", home)
	}()

	_ = os.Setenv("HOME", abs)
	if err := os.Mkdir(filepath.Join(abs, ".cache"), 0777); err != nil {
		t.Fatal("Could not initialize test")
	}

	file := cachePath()
	if file == nil {
		t.Fatal("The given file should be ok")
	}
	info, err := file.Stat()
	if err != nil {
		t.Fatal("I should be able to stat the given file")
	}
	name, mode := file.Name(), info.Mode().Perm()
	_ = file.Close()
	_ = os.Remove(name)
	if name != filepath.Join(abs, ".cache", cacheName) {
		t.Fatal("Unexpected name")
	}
	if mode != 0666 {
		t.Fatal("Given file does not come from hell ;)")
	}
}

func TestCachePathFail(t *testing.T) {
	home, umask := os.Getenv("HOME"), syscall.Umask(0)

	defer func() {
		syscall.Umask(umask)
		_ = os.Setenv("HOME", home)
	}()

	_ = os.Setenv("HOME", "")
	file, _ := os.OpenFile(filepath.Join("/tmp", cacheName), os.O_RDONLY|os.O_CREATE, 0000)
	_ = file.Close()

	buffer := bytes.NewBuffer([]byte{})
	log.SetOutput(buffer)
	cacheFile := getCacheFile()
	if cacheFile.Valid {
		t.Fatal("Cache should not be valid")
	}
	if !strings.Contains(buffer.String(), "could not find path for the cache") {
		t.Fatal("Wrong log")
	}
}

func TestCacheBadJson(t *testing.T) {
	home, umask := os.Getenv("HOME"), syscall.Umask(0)
	abs, _ := filepath.Abs(".")
	test := filepath.Join(abs, "test")

	defer func() {
		syscall.Umask(umask)
		_ = os.Setenv("HOME", home)
		_ = os.Rename(filepath.Join(test, ".cache", cacheName),
			filepath.Join(test, ".cache", "bad.json"))
	}()

	_ = os.Setenv("HOME", test)
	_ = os.Rename(filepath.Join(test, ".cache", "bad.json"),
		filepath.Join(test, ".cache", cacheName))

	buffer := bytes.NewBuffer([]byte{})
	log.SetOutput(buffer)
	file := getCacheFile()
	if !file.Valid {
		t.Fatal("It should be valid")
	}
	if !strings.Contains(buffer.String(), "decoding of cache file failed") {
		t.Fatal("Wrong log")
	}
}

func TestCacheGoodJson(t *testing.T) {
	home, umask := os.Getenv("HOME"), syscall.Umask(0)
	abs, _ := filepath.Abs(".")
	test := filepath.Join(abs, "test")

	defer func() {
		syscall.Umask(umask)
		_ = os.Setenv("HOME", home)
		_ = os.Rename(filepath.Join(test, ".cache", cacheName),
			filepath.Join(test, ".cache", "ok.json"))
	}()

	_ = os.Setenv("HOME", test)
	_ = os.Rename(filepath.Join(test, ".cache", "ok.json"),
		filepath.Join(test, ".cache", cacheName))

	file := getCacheFile()

	if !file.Valid {
		t.Fatal("It should be valid")
	}
	if file.Path != filepath.Join(test, ".cache", cacheName) {
		t.Fatal("Wrong path")
	}

	testSlice(t, file.IDs["zypper"], []string{"1", "2"})
	testSlice(t, file.IDs["dnf"], []string{"4"})
	testSlice(t, file.IDs[otherKey], []string{"3", "5", "6"})
	testSlice(t, file.Outdated, []string{})
}

func TestFlush(t *testing.T) {
	abs, _ := filepath.Abs(".")
	test := filepath.Join(abs, "test")
	path := filepath.Join(test, "testflush.json")

	cd := &cachedData{
		Path:  path,
		Valid: false,
		IDs:   make(map[string][]string),
	}

	// Now put some contents there.
	expected := "{\"ids\":{\"dnf\":[\"1\"],\"others\":[],\"zypper\":[]},\"outdated\":[]}"
	err := ioutil.WriteFile(path, []byte(expected), 0666)
	if err != nil {
		t.Fatal("Failed on writing a file")
	}

	// It's invalid, flush will do nothing.
	cd.flush()

	contents, err := ioutil.ReadFile(path)
	if err != nil {
		t.Fatal("Failed on reading a file")
	}
	if strings.TrimSpace(string(contents)) != expected {
		t.Fatal("Wrong contents")
	}

	// Now it will re-read the content of the file before writing to it
	// again.
	cd.Valid = true
	cd.flush()
	contents, err = ioutil.ReadFile(path)
	if err != nil {
		t.Fatal("Failed on reading a file")
	}
	if strings.TrimSpace(string(contents)) != expected {
		fmt.Println("got", string(contents), "instead of", expected)
		t.Fatal("Wrong contents")
	}

	// If we remove the file and try to access it, it will print a proper
	// error.
	buffer := bytes.NewBuffer([]byte{})
	log.SetOutput(buffer)
	if err := os.Remove(path); err != nil {
		t.Fatal("Could not remove temporary file")
	}
	cd.flush()
	if !strings.Contains(buffer.String(), "cannot write to the cache file") {
		t.Fatal("Didn't logged what it was expected")
	}
}

func TestUpdateCacheAfterUpdateFailsBecauseOfListError(t *testing.T) {
	cache := cachedData{}

	safeClient.client = &mockClient{listFail: true}
	err := cache.updateCacheAfterUpdate("1", "2", "zypper")
	if err == nil {
		t.Fatal("Expected failure")
	}
}

func TestUpdateCacheAfterUpdateFailsBecauseOfListEmpty(t *testing.T) {
	cache := cachedData{}

	safeClient.client = &mockClient{listEmpty: true}
	err := cache.updateCacheAfterUpdate("1", "2", "zypper")
	if err == nil {
		t.Fatal("Expected failure")
	}
}

func TestUpdateCacheAfterUpdateNothingDoneWhenTheImageIsAlreadyKnown(t *testing.T) {
	cache := cachedData{
		Outdated: []string{"35ae93c88cf8ab18da63bb2ad2dfd2399d745f292a344625fbb65892b7c25a01"},
		IDs:      map[string][]string{"zypper": []string{"2"}},
	}

	safeClient.client = &mockClient{listEmpty: true}
	err := cache.updateCacheAfterUpdate("opensuse:13.2", "2", "zypper")
	if err == nil {
		t.Fatal("Expected failure")
	}
	if len(cache.Outdated) != 1 {
		t.Fatal("Nothing should have changed")
	}
	if len(cache.IDs["zypper"]) != 1 {
		t.Fatal("Nothing should have changed")
	}
}

func TestReadCacheSuccess(t *testing.T) {
	cache := cachedData{}
	expected := &cachedData{
		Valid: true,
		IDs: map[string][]string{
			"zypper": []string{"1"},
			"others": []string{"2"},
		},
		Outdated: []string{"3"},
	}
	buffer := bytes.NewBufferString(`{"ids":{"zypper":["1"],"others":["2"]},"outdated":["3"]}`)

	got := cache.readCache(buffer)
	if !reflect.DeepEqual(got, expected) {
		t.Fatalf("Expected %v, got %v", expected, got)
	}
}
