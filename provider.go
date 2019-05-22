package fifo

import (
	"github.com/pkg/errors"
	"github.com/rlmcpherson/s3gof3r"
	"io"
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

type DestroyableS3Object struct {
	Path   string
	Bucket *s3gof3r.Bucket
	io.WriteCloser
}

func (o *DestroyableS3Object) Destroy() error {
	return o.Bucket.Delete(o.Path)
}

// Provides files from an S3-like HTTP interface
type S3Provider struct {
	AccessKeyID    string
	SecretKey      string
	Endpoint       string
	PathAddressing bool
}

func (S3Provider) Schema() []string {
	return []string{"s3", "s3+insecure"}
}

func (p S3Provider) Bucket(u *url.URL) *s3gof3r.Bucket {
	s3 := s3gof3r.New(p.Endpoint, s3gof3r.Keys{
		AccessKey: p.AccessKeyID,
		SecretKey: p.SecretKey,
	})

	c := new(s3gof3r.Config)
	*c = *s3gof3r.DefaultConfig
	c.PathStyle = p.PathAddressing
	c.Md5Check = false

	if u.Scheme == "s3+insecure" {
		c.Scheme = "http"
	}

	b := s3.Bucket(u.Host)
	b.Config = c

	return b
}

func (p S3Provider) Read(u *url.URL) (io.ReadCloser, error) {
	b := p.Bucket(u)
	r, _, err := b.GetReader(u.Path, b.Config)
	return r, err
}

func (p S3Provider) Write(u *url.URL) (WriteDestroyCloser, error) {
	b := p.Bucket(u)
	w, err := b.PutWriter(u.Path, nil, nil)
	if err != nil {
		return nil, err
	}
	return &DestroyableS3Object{
		Path:        u.Path,
		Bucket:      b,
		WriteCloser: w,
	}, nil
}
