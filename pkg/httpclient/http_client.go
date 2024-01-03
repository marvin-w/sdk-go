package httpclient

import (
	"net/http"
	"time"
)

// Requester exposes the http.Client.Do method, which is the minimum
// required method for executing HTTP requests.
type Requester interface {
	Do(*http.Request) (*http.Response, error)
}

type clientOptions struct {
	Timeout time.Duration
}

type retryOptions struct {
	clientOptions
	BackoffStrategy BackoffFunc
	CheckRetry      CheckRetryFunc
}

// Option signature for client configurable parameters.
type Option interface {
	OptionRetryable
	applyClient(opts *clientOptions)
}

// OptionRetryable signature for retryable client configurable parameters.
type OptionRetryable interface {
	applyRetryable(opts *retryOptions)
}

type optFunc func(opts *clientOptions)

func (f optFunc) applyClient(o *clientOptions)   { f(o) }
func (f optFunc) applyRetryable(o *retryOptions) { f(&o.clientOptions) }

type retryableOptFunc func(opts *retryOptions)

func (f retryableOptFunc) applyRetryable(o *retryOptions) { f(o) }

// WithTimeout controls the timeout for each request. When retrying requests,
// each retried request will start counting from the beginning towards this
// timeout.
//
// A timeout of 0 disables request timeouts.
func WithTimeout(t time.Duration) Option {
	return optFunc(func(options *clientOptions) {
		// Negative durations do not make sense in the context of an Requester.
		if t >= 0 {
			options.Timeout = t
		}
	})
}

// WithBackoffStrategy controls the wait time between requests when retrying.
func WithBackoffStrategy(strategy BackoffFunc) OptionRetryable {
	return retryableOptFunc(func(options *retryOptions) {
		options.BackoffStrategy = strategy
	})
}

// WithRetryPolicy controls the retry policy of the given HTTP client.
func WithRetryPolicy(checkRetry CheckRetryFunc) OptionRetryable {
	return retryableOptFunc(func(options *retryOptions) {
		options.CheckRetry = checkRetry
	})
}

var (
	// DefaultTimeout is the timeout used by default when building a Client.
	DefaultTimeout = 10 * time.Second

	// DefaultBackoffStrategy is the retry strategy used by default when
	// building a Client.
	DefaultBackoffStrategy = ConstantBackoff(0)

	// DefaultRetryPolicy is the function that tells on any given request if the
	// client should retry it or not. By default, it retries on connection and 5xx errors only.
	DefaultRetryPolicy = ServerErrorsRetryPolicy()
)

// New builds a *http.Client which keeps TCP connections to destination servers.
//
// Returned client can be customized by passing options to New.
func New(opts ...Option) *http.Client {
	config := clientOptions{
		Timeout: DefaultTimeout,
	}

	for _, opt := range opts {
		opt.applyClient(&config)
	}

	return &http.Client{
		Timeout: config.Timeout,
	}
}

// NewRetryable builds a *RetryableClient which keeps TCP connections to
// destination servers, can retry requests on error.
//
// RetryableClient can be customized by passing options to it. Note that Option
// is of type OptionRetryable, so those functional options can be used as well.
//
// RetryMax tells the client the maximum number of retries to execute. Eg.: A
// value of 3, means to execute the original request, and up-to 3 retries (4
// requests in total). A value of 0 means no retries, essentially the same as
// building a *http.Client with New.
func NewRetryable(retryMax int, opts ...OptionRetryable) *RetryableClient {
	config := retryOptions{
		BackoffStrategy: DefaultBackoffStrategy,
		CheckRetry:      DefaultRetryPolicy,
		clientOptions: clientOptions{
			Timeout: DefaultTimeout,
		},
	}

	for _, opt := range opts {
		opt.applyRetryable(&config)
	}

	return &RetryableClient{
		RetryMax:        retryMax,
		BackoffStrategy: config.BackoffStrategy,
		CheckRetry:      config.CheckRetry,
		Client: &http.Client{
			Timeout: config.Timeout,
		},
	}
}
