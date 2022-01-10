# multipart

Made on the basis of mime/multipart

## Example

```go
package main

import (
	"context"
	"net/http"
	"time"

	"github.com/GoWebProd/multipart"
	"github.com/pkg/errors"
)

func main() {
	fileReq, err := http.NewRequestWithContext(context.Background(), http.MethodGet, "http://localhost/testfile", nil)
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

	writer := multipart.NewWriter()

	err = writer.CreateFormFileReader("content", "very_important_file_name", fileResp.Body)
	if err != nil {
		panic(errors.Wrap(err, "Can't create file form"))
	}

	err = writer.CreateFormField("objectType", []byte("file"))
	if err != nil {
		panic(errors.Wrap(err, "Can't write object type field"))
	}

	req, err := http.NewRequestWithContext(context.Background(), http.MethodPost, "http://localhost/checkfile", writer)
	if err != nil {
		panic(errors.Wrap(err, "Can't create object scan request"))
	}

	req.Header.Add("Content-Type", writer.FormDataContentType())

	resp, err := httpClient.Do(req)
	if err != nil {
		panic(errors.Wrap(err, "Can't do object scan request"))
	}

	defer resp.Body.Close()
}

```

## Benchmark

```
goos: darwin
goarch: amd64
pkg: github.com/GoWebProd/multipart
BenchmarkStd-16           465567              2602 ns/op            1345 B/op         37 allocs/op
BenchmarkThis-16         1557962               755 ns/op             275 B/op          7 allocs/op
PASS
ok      github.com/GoWebProd/multipart  4.510s
```
