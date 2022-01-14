package multipart

import (
	"bytes"
	"crypto/rand"
	"errors"
	"io"
	"strings"
)

type Reader interface {
	io.Reader

	Len() int
}

type header struct {
	key   string
	value string
}

type part struct {
	headers []header
	body    Reader
}

// A Writer generates multipart messages.
type Writer struct {
	boundary []byte
	parts    []part

	writePosition int
	sysBuf        bytes.Buffer
}

// NewWriter returns a new multipart Writer with a random boundary.
func NewWriter() *Writer {
	w := &Writer{
		boundary:      make([]byte, 60),
		parts:         make([]part, 0, 8),
		writePosition: -1,
	}

	w.randomBoundary()

	return w
}

func (w *Writer) Reset() {
	w.randomBoundary()
	w.parts = w.parts[:0]
	w.writePosition = -1

	w.sysBuf.Reset()
}

// Boundary returns the Writer's boundary.
func (w *Writer) Boundary() []byte {
	return w.boundary
}

// SetBoundary overrides the Writer's default randomly-generated
// boundary separator with an explicit value.
//
// SetBoundary must be called before any parts are created, may only
// contain certain ASCII characters, and must be non-empty and
// at most 70 bytes long.
func (w *Writer) SetBoundary(boundary []byte) error {
	// rfc2046#section-5.1.1
	if len(boundary) < 1 || len(boundary) > 70 {
		return errors.New("mime: invalid boundary length")
	}

	end := len(boundary) - 1

	for i, b := range boundary {
		if 'A' <= b && b <= 'Z' || 'a' <= b && b <= 'z' || '0' <= b && b <= '9' {
			continue
		}

		switch b {
		case '\'', '(', ')', '+', '_', ',', '-', '.', '/', ':', '=', '?':
			continue
		case ' ':
			if i != end {
				continue
			}
		}

		return errors.New("mime: invalid boundary character")
	}

	w.boundary = boundary

	return nil
}

// FormDataContentType returns the Content-Type for an HTTP
// multipart/form-data with this Writer's Boundary.
func (w *Writer) FormDataContentType() string {
	b := string(w.boundary)

	// We must quote the boundary if it contains any of the
	// tspecials characters defined by RFC 2045, or space.
	if strings.ContainsAny(b, `()<>@,;:\"/[]?= `) {
		b = `"` + b + `"`
	}

	return "multipart/form-data; boundary=" + b
}

const hextable = "0123456789abcdef"

func (w *Writer) randomBoundary() {
	_, err := io.ReadFull(rand.Reader, w.boundary[:30])
	if err != nil {
		panic(err)
	}

	j := len(w.boundary) - 2

	for i := 29; i >= 0; i-- {
		v := w.boundary[i]

		w.boundary[j] = hextable[v>>4]
		w.boundary[j+1] = hextable[v&0x0f]

		j -= 2
	}
}

var quoteEscaper = strings.NewReplacer("\\", "\\\\", `"`, "\\\"")

func escapeQuotes(s string) string {
	return quoteEscaper.Replace(s)
}

func (w *Writer) createPart(p part) error {
	if w.writePosition >= 0 {
		return errors.New("mime: create called after write")
	}

	w.parts = append(w.parts, p)

	return nil
}

// CreateFormFile creates a new form-data header with
// the provided field name and file name and data.
func (w *Writer) CreateFormFile(fieldname string, filename string, data []byte) error {
	h := []header{
		{"Content-Disposition", `form-data; name="` + escapeQuotes(fieldname) + `"; filename="` + escapeQuotes(filename) + `"`},
		{"Content-Type", "application/octet-stream"},
	}

	return w.createPart(part{
		headers: h,
		body:    bytes.NewReader(data),
	})
}

// CreateFormFileReader creates a new form-data header with
// the provided field name and file name and reader.
func (w *Writer) CreateFormFileReader(fieldname string, filename string, data Reader) error {
	h := []header{
		{"Content-Disposition", `form-data; name="` + escapeQuotes(fieldname) + `"; filename="` + escapeQuotes(filename) + `"`},
		{"Content-Type", "application/octet-stream"},
	}

	return w.createPart(part{
		headers: h,
		body:    data,
	})
}

// CreateFormField creates part with a header using the
// given field name and data.
func (w *Writer) CreateFormField(fieldname string, data []byte) error {
	h := []header{
		{"Content-Disposition", `form-data; name="` + escapeQuotes(fieldname) + `"`},
	}

	return w.createPart(part{
		headers: h,
		body:    bytes.NewReader(data),
	})
}

// CreateFormFieldReader creates part with a header using the
// given field name and reader.
func (w *Writer) CreateFormFieldReader(fieldname string, data Reader) error {
	h := []header{
		{"Content-Disposition", `form-data; name="` + escapeQuotes(fieldname) + `"`},
	}

	return w.createPart(part{
		headers: h,
		body:    data,
	})
}

func (w *Writer) Read(dst []byte) (int, error) {
	for {
		if w.sysBuf.Len() > 0 {
			return w.sysBuf.Read(dst)
		}

		if w.writePosition >= len(w.parts) {
			return 0, io.EOF
		}

		if w.writePosition != -1 {
			n, err := w.parts[w.writePosition].body.Read(dst)
			if n != 0 {
				return n, nil
			}

			if !errors.Is(err, io.EOF) {
				return 0, err
			}

			w.sysBuf.Reset()
		}

		w.writePosition++

		if w.writePosition == len(w.parts) {
			w.sysBuf.WriteString("\r\n--")
			w.sysBuf.Write(w.boundary)
			w.sysBuf.WriteString("--\r\n")

			continue
		}

		if w.writePosition != 0 {
			w.sysBuf.WriteString("\r\n")
		}
		w.sysBuf.WriteString("--")
		w.sysBuf.Write(w.boundary)
		w.sysBuf.WriteString("\r\n")

		p := w.parts[w.writePosition]

		for _, p := range p.headers {
			w.sysBuf.WriteString(p.key)
			w.sysBuf.WriteString(": ")
			w.sysBuf.WriteString(p.value)
			w.sysBuf.WriteString("\r\n")
		}

		w.sysBuf.WriteString("\r\n")
	}
}

func (w *Writer) Close() error {
	return nil
}

func (w *Writer) Len() int {
	l := 0

	for _, v := range w.parts {
		l += 4 + len(w.boundary)

		for _, p := range v.headers {
			l += 4 + len(p.key) + len(p.value)
		}

		l += 4 + v.body.Len()
	}

	if l > 0 {
		l += 6 + len(w.boundary)
	}

	return l
}

type readSizer struct {
	r io.Reader
	l int
}

func NewReader(r io.Reader, length int) Reader {
	return &readSizer{
		r: r,
		l: length,
	}
}

func (r readSizer) Read(b []byte) (int, error) {
	return r.r.Read(b)
}

func (r readSizer) Len() int {
	return r.l
}
