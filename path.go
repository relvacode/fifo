package fifo

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/valyala/fasttemplate"
	"io"
	"math/rand"
	"net/url"
	"os"
	"strconv"
	"strings"
	"time"
)

var functions = map[string]func() string{
	"date": func() string {
		return time.Now().Format("2006-01-02")
	},
	"time": func() string {
		return time.Now().Format("15:04:05")
	},
	"datetime": func() string {
		return time.Now().Format("2006-01-02T15:04:05")
	},
	"hostname": func() string {
		host, _ := os.Hostname()
		return host
	},
	"uid": func() string {
		return strconv.Itoa(os.Getuid())
	},
	"gid": func() string {
		return strconv.Itoa(os.Getgid())
	},
	"random": func() string {
		return fmt.Sprintf("%06d", rand.Intn(999999))
	},
}

type Url url.URL

func (f *Url) provide(w io.Writer, tag string) (int, error) {
	fn, ok := functions[strings.TrimSpace(tag)]
	if !ok {
		return 0, errors.Errorf("No built-in template function with name %q", tag)
	}

	v := url.PathEscape(fn())
	return fmt.Fprint(w, v)
}

func (f *Url) replace(arg string) (string, error) {
	t := fasttemplate.New(arg, "@{", "}")
	var b bytes.Buffer
	_, err := t.ExecuteFunc(&b, f.provide)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

func (f *Url) UnmarshalFlag(value string) error {
	rendered, err := f.replace(value)
	if err != nil {
		return err
	}
	u, err := url.Parse(rendered)
	if err != nil {
		return err
	}
	if u.Scheme == "" {
		return errors.New("a url scheme is required but was not provided")
	}
	*f = *(*Url)(u)
	return nil
}

type UrlMapping map[string]*Url

// UnmarshalFlag implements un-marshalling a flag value into the URL mapping.
// Where the format is key:url
func (m UrlMapping) UnmarshalFlag(value string) error {
	parts := strings.Split(value, "=")
	if len(parts) < 2 {
		return errors.Errorf("expected tag=url format of flag")
	}

	u := new(Url)
	err := u.UnmarshalFlag(strings.Join(parts[1:], ":"))
	if err != nil {
		return err
	}

	m[parts[0]] = u
	return nil
}

type runeWriter interface {
	WriteRune(r rune) (n int, err error)
}

var filenameRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ01234567890_-. ")
var extensionRunes = []rune("abcdefghijklmnopqrstuvwxyz0123456789")

func randExtensionString(w runeWriter, n int) {
	for i := 0; i < n; i++ {
		_, _ = w.WriteRune(extensionRunes[rand.Intn(len(extensionRunes))])
	}
	_, _ = w.WriteRune('.')
}

// urlToFilename sanitises a URL into a name compatible with any file system.
// Specifically, any characters not a member of `filenameRunes` is replaced with `_`
func urlToFilename(u *url.URL) string {
	var s = u.String()
	var b bytes.Buffer
	randExtensionString(&b, 6)

scan:
	for _, r := range s {
		for _, xr := range filenameRunes {
			if r == xr {
				b.WriteRune(r)
				continue scan
			}
		}
		b.WriteRune('_')
	}
	return b.String()
}
