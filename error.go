package fifo

import "fmt"

func Catch(mu *MultiError, err ...error) *MultiError {
	if err == nil || len(err) == 0 {
		return nil
	}
	if mu == nil {
		mu = new(MultiError)
	}

	var containsNonNilError bool
	for _, e := range err {
		if e != nil {
			containsNonNilError = true
		}
		mu.Append(e)
	}
	if !containsNonNilError {
		return nil
	}
	return mu
}

type MultiError struct {
	err []error
}

func (mu *MultiError) Append(err error) {
	switch t := err.(type) {
	case nil:
		return
	case *MultiError:
		if t != nil {
			mu.err = append(mu.err, t.err...)
		}
	case error:
		mu.err = append(mu.err, err)
	}
}

func (mu *MultiError) Catch(funcs ...func() error) (failure bool) {
	for _, f := range funcs {
		err := f()
		if err != nil {
			failure = true
			mu.Append(err)
		}
	}
	return
}

type ErrorFunc func() *MultiError

func (mu *MultiError) CatchMulti(funcs ...ErrorFunc) (failure bool) {
	for _, f := range funcs {
		err := f()
		if err != nil && len(err.err) > 0 {
			mu.Append(err)
			failure = true
		}
	}

	return
}

func (mu *MultiError) Error() string {
	return fmt.Sprintf("(%d) errors", len(mu.err))
}

func (mu *MultiError) Errors() []error {
	if mu == nil || len(mu.err) == 0 {
		return nil
	}
	return mu.err
}

func (mu *MultiError) AsError() error {
	if mu == nil || len(mu.err) == 0 {
		return nil
	}
	return mu
}
