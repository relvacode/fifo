package fifo

import (
	"bytes"
	"fmt"
	"github.com/pkg/errors"
	"github.com/valyala/fasttemplate"
	"io"
	"net/url"
	"strings"
)

// A PipeProvider is given a URL and should return a Source or Target pipe.
type PipeProvider interface {
	Source(u *url.URL) (*SourcePipe, error)
	Target(u *url.URL) (*TargetPipe, error)
}

// A TemplateGenerator replaces command-line argument values with a real location of a fifo on the file-system
type TemplateGenerator struct {
	Provider PipeProvider

	SourceTags UrlMapping
	Sources    Sources

	TargetTags UrlMapping
	Targets    Targets
}

func (g *TemplateGenerator) provide(w io.Writer, tag string) (int, error) {
	tag = strings.TrimSpace(tag)
	st, sok := g.SourceTags[tag]
	tt, tok := g.TargetTags[tag]
	if sok && tok {
		return 0, errors.Errorf("tag %q described as both source and target", tag)
	}
	if !sok && !tok {
		return 0, errors.Errorf("tag %q not defined in sources or targets", tag)
	}

	if sok {
		p, err := g.Provider.Source((*url.URL)(st))
		if err != nil {
			return 0, err
		}
		g.Sources = append(g.Sources, p)
		return fmt.Fprint(w, p.Path)

	} else {
		p, err := g.Provider.Target((*url.URL)(tt))
		if err != nil {
			return 0, err
		}
		g.Targets = append(g.Targets, p)
		return fmt.Fprint(w, p.Path)
	}
}

func (g *TemplateGenerator) replaceArgument(arg string) (string, error) {
	t := fasttemplate.New(arg, "%{", "}")
	var b bytes.Buffer
	_, err := t.ExecuteFunc(&b, g.provide)
	if err != nil {
		return "", err
	}

	return b.String(), nil
}

// Replace replaces the contents of args with the templated values found from the given source and target mappings.
// returns all un-used source and target mappings.
func (g *TemplateGenerator) Replace(args []string) ([]string, error) {
	compiled := make([]string, len(args))
	for i, a := range args {
		str, err := g.replaceArgument(a)
		if err != nil {
			return nil, errors.Wrapf(err, "failed to compile argument at position %d", i)
		}
		compiled[i] = str
	}

	return compiled, nil
}
