package multipart

import (
	"bytes"
	"context"
	"io"
	"mime/multipart"
	"net/http"
	"testing"
	"time"

	"github.com/google/uuid"
	"github.com/pkg/errors"
)

func TestCompare(t *testing.T) {
	fileContents := []byte("my file contents")

	var b1 bytes.Buffer

	w1 := multipart.NewWriter(&b1)

	part, err := w1.CreateFormFile("myfile", "my-file.txt")
	if err != nil {
		t.Fatalf("CreateFormFile: %v", err)
	}

	part.Write(fileContents)

	err = w1.WriteField("key", "val")
	if err != nil {
		t.Fatalf("WriteField: %v", err)
	}

	err = w1.Close()
	if err != nil {
		t.Fatalf("Close: %v", err)
	}

	w2 := NewWriter()

	w2.SetBoundary([]byte(w1.Boundary()))
	w2.CreateFormFileReader("myfile", "my-file.txt", bytes.NewReader(fileContents))
	w2.CreateFormField("key", []byte("val"))

	var b2 bytes.Buffer

	_, err = io.Copy(&b2, w2)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if !bytes.Equal(b2.Bytes(), b1.Bytes()) {
		t.Logf("b1: %v", b1.String())
		t.Logf("b2: %v", b2.String())
		t.Fatal("b1 != b2")
	}
}

func TestEicar(t *testing.T) {
	fileReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "https://secure.eicar.org/eicar.com", nil)
	if err != nil {
		panic(errors.Wrap(err, "Can't create new request"))
	}

	httpClient := &http.Client{
		Timeout: 5 * time.Second,
	}

	fileResp, err := httpClient.Do(fileReq)
	if err != nil {
		panic(errors.Wrap(err, "Can't file request"))
	}

	defer fileResp.Body.Close()

	w2 := NewWriter()
	scanID, _ := uuid.NewUUID()

	err = w2.CreateFormFileReader("myfile", "my-file.txt", NewReader(fileResp.Body, int(fileResp.ContentLength)))
	if err != nil {
		panic(err)
	}

	err = w2.CreateFormField("scanId", []byte(scanID.String()))
	if err != nil {
		panic(err)
	}

	err = w2.CreateFormField("objectType", []byte("file"))
	if err != nil {
		panic(err)
	}

	length := w2.Len()

	var b2 bytes.Buffer

	_, err = io.Copy(&b2, w2)
	if err != nil {
		t.Fatalf("Copy: %v", err)
	}

	if b2.Len() != length {
		t.Logf("b2: %v", b2.String())
		t.Fatalf("b2.Len() != w2.Len(): %d != %d", b2.Len(), length)
	}
}

func BenchmarkStd(b *testing.B) {
	b1 := bytes.NewBuffer(nil)

	for i := 0; i < b.N; i++ {
		fileContents := []byte("my file contents")

		b1.Reset()

		w1 := multipart.NewWriter(b1)

		part, err := w1.CreateFormFile("myfile", "my-file.txt")
		if err != nil {
			b.Fatalf("CreateFormFile: %v", err)
		}

		part.Write(fileContents)

		err = w1.WriteField("key", "val")
		if err != nil {
			b.Fatalf("WriteField: %v", err)
		}

		part.Write([]byte("val"))

		err = w1.Close()
		if err != nil {
			b.Fatalf("Close: %v", err)
		}
	}
}

func BenchmarkThis(b *testing.B) {
	w2 := NewWriter()
	b2 := bytes.NewBuffer(nil)
	fileContents := []byte("my file contents")

	for i := 0; i < b.N; i++ {
		b2.Reset()
		w2.Reset()

		w2.CreateFormFile("myfile", "my-file.txt", fileContents)
		w2.CreateFormField("key", []byte("val"))

		_, err := io.Copy(b2, w2)
		if err != nil {
			b.Fatalf("Copy: %v", err)
		}
	}
}
