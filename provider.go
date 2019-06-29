package fifo

import (
	"context"
	"github.com/aws/aws-sdk-go/aws"
	"github.com/aws/aws-sdk-go/aws/session"
	"github.com/aws/aws-sdk-go/service/s3"
	"github.com/aws/aws-sdk-go/service/s3/s3manager"
	"github.com/pkg/errors"
	"golang.org/x/sync/errgroup"
	"io"
	"net/http"
	"net/url"
	"os"
	"path/filepath"
)

type Provider interface {
	Schema() []string
}

type SourceProvider interface {
	Provider
	Read(*url.URL) (io.ReadCloser, error)
}

type WriteDestroyCloser interface {
	io.WriteCloser
	// Teardown is called when the command fails, signalling that the object should be removed.
	// Closer must be closed before calling abort.
	Destroy() error
}

type TargetProvider interface {
	Provider
	Write(*url.URL) (WriteDestroyCloser, error)
}

// ProvideSource opens a source stream for a given URN from a given list of providers.
func ProvideSource(u *url.URL, providers ...Provider) (io.ReadCloser, error) {
	for _, p := range providers {
		sp, ok := p.(SourceProvider)
		if !ok {
			continue
		}
		for _, s := range p.Schema() {
			if s == u.Scheme {
				return sp.Read(u)
			}
		}
	}

	return nil, errors.Errorf("no such source provider for scheme %q", u.Scheme)
}

func ProvideTarget(u *url.URL, providers ...Provider) (WriteDestroyCloser, error) {
	for _, p := range providers {
		tp, ok := p.(TargetProvider)
		if !ok {
			continue
		}
		for _, s := range p.Schema() {
			if s == u.Scheme {
				return tp.Write(u)
			}
		}
	}

	return nil, errors.Errorf("no such target provider for scheme %q", u.Scheme)
}

type DestroyableFile struct {
	Path string
	*os.File
}

func (f *DestroyableFile) Destroy() error {
	return os.Remove(f.Path)
}

type FileProvider struct {
	Create os.FileMode
}

func (FileProvider) Schema() []string {
	return []string{"file"}
}

func (FileProvider) Target(u *url.URL) string {
	return filepath.Join(u.Host, u.Path)
}

func (fp FileProvider) Read(u *url.URL) (io.ReadCloser, error) {
	return os.OpenFile(fp.Target(u), os.O_RDONLY, os.FileMode(0600))
}

func (fp FileProvider) Write(u *url.URL) (WriteDestroyCloser, error) {
	path := fp.Target(u)
	f, err := os.OpenFile(path, os.O_WRONLY|os.O_CREATE, fp.Create)
	if err != nil {
		return nil, err
	}

	return &DestroyableFile{
		Path: path,
		File: f,
	}, nil
}

type S3PutObject struct {
	s *session.Session
	g *errgroup.Group

	key    string
	bucket string

	pr io.ReadCloser
	pw io.WriteCloser
}

func (o *S3PutObject) Write(b []byte) (int, error) {
	return o.pw.Write(b)
}

func (o *S3PutObject) Close() error {
	mu := new(MultiError)
	mu.Catch(o.pw.Close)
	mu.Append(o.g.Wait())
	return mu.AsError()
}

func (o *S3PutObject) Destroy() error {
	_, err := s3.New(o.s).DeleteObject(&s3.DeleteObjectInput{
		Key:    aws.String(o.key),
		Bucket: aws.String(o.bucket),
	})
	return err
}

// Provides files from an S3-like HTTP interface
type S3Provider struct {
	Endpoint string
	Region   string
}

func (S3Provider) Schema() []string {
	return []string{"s3", "s3+insecure"}
}

func (p S3Provider) Session(u *url.URL) (*session.Session, error) {
	var disableSsl bool
	if u.Scheme == "s3+insecure" {
		disableSsl = true
	}
	return session.NewSession(&aws.Config{
		Endpoint:   aws.String(p.Endpoint),
		Region:     aws.String(p.Region),
		DisableSSL: &disableSsl,
	})
}

func (p S3Provider) Read(u *url.URL) (io.ReadCloser, error) {
	s, err := p.Session(u)
	if err != nil {
		return nil, err
	}

	o, err := s3.New(s).GetObject(&s3.GetObjectInput{
		Bucket: aws.String(u.Host),
		Key:    aws.String(u.Path),
	})

	if err != nil {
		return nil, err
	}

	return o.Body, nil
}

func (p S3Provider) uploadInput(u *url.URL, r io.Reader) *s3manager.UploadInput {
	q := u.Query()
	return &s3manager.UploadInput{
		Bucket:      aws.String(u.Host),
		Key:         aws.String(u.Path),
		ACL:         aws.String(q.Get("acl")),
		ContentType: aws.String(q.Get("type")),
		Body:        r,
	}
}

func (p S3Provider) Write(u *url.URL) (WriteDestroyCloser, error) {
	s, err := p.Session(u)
	if err != nil {
		return nil, err
	}

	pr, pw := io.Pipe()

	uploader := s3manager.NewUploader(s)

	g, ctx := errgroup.WithContext(context.Background())
	g.Go(func() error {
		_, err := uploader.UploadWithContext(ctx, p.uploadInput(u, pr))
		return Catch(nil, err, pr.Close()).AsError()
	})

	return &S3PutObject{
		s:      s,
		g:      g,
		key:    u.Path,
		bucket: u.Host,
		pr:     pr,
		pw:     pw,
	}, nil
}

// HTTPProvider provides a source from an HTTP url
type HTTPProvider struct {
	Client *http.Client
}

func (p *HTTPProvider) Schema() []string {
	return []string{"http", "https"}
}

func (p *HTTPProvider) Read(u *url.URL) (io.ReadCloser, error) {
	req, err := http.NewRequest(http.MethodGet, u.String(), nil)
	if err != nil {
		return nil, err
	}

	req.Header.Set("User-Agent", "fifo/0.1a")

	resp, err := p.Client.Do(req)
	if err != nil {
		return nil, err
	}

	return resp.Body, nil
}
