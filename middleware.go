package caddyawslambda

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"net/http"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/caddyserver/caddy/v2"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

var (
	_ caddy.Module                = (*LambdaMiddleware)(nil)
	_ caddy.Provisioner           = (*LambdaMiddleware)(nil)
	_ caddy.Validator             = (*LambdaMiddleware)(nil)
	_ caddyhttp.MiddlewareHandler = (*LambdaMiddleware)(nil)

	defaultMeta = ReplyMeta{
		Status: 200,
		Headers: map[string][]string{
			"content-type": {"application/json"},
		},
	}
)

func init() {
	caddy.RegisterModule(&LambdaMiddleware{})
}

// LambdaMiddleware implements an HTTP handler that invokes a lambda function
type LambdaMiddleware struct {
	FunctionName string `json:"function,omitempty"`
	Timeout      string `json:"timeout,omitempty"`

	timeout time.Duration
	log     *zap.Logger
	svc     *lambda.Client
}

// CaddyModule returns the Caddy module information.
func (*LambdaMiddleware) CaddyModule() caddy.ModuleInfo {
	return caddy.ModuleInfo{
		ID:  "http.handlers.awslambda",
		New: func() caddy.Module { return &LambdaMiddleware{} },
	}
}

// Provision implements caddy.Provisioner.
func (m *LambdaMiddleware) Provision(ctx caddy.Context) error {
	m.log = ctx.Logger(m)

	if m.Timeout == "" {
		m.Timeout = "10s"
	}

	dur, err := time.ParseDuration(m.Timeout)
	if err != nil {
		return fmt.Errorf("Invalid value for timeout: %w", err)
	}
	m.timeout = dur

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("Unable to load AWS config: %w", err)
	}

	m.svc = lambda.NewFromConfig(cfg)

	return nil
}

// Validate implements caddy.Validator
func (m *LambdaMiddleware) Validate() error {
	return nil
}

// ServeHTTP implements caddyhttp.MiddlewareHandler.
func (m *LambdaMiddleware) ServeHTTP(w http.ResponseWriter, r *http.Request, next caddyhttp.Handler) error {
	req, err := newRequest(r)
	if err != nil {
		return err
	}

	resp, err := m.invokeLambda(r.Context(), req)

	if err != nil {
		return err
	}

	// Unpack the reply JSON
	reply, err := parseReply(resp)
	if err != nil {
		return err
	}

	// Write the response HTTP headers
	for k, vals := range reply.Meta.Headers {
		for _, v := range vals {
			w.Header().Add(k, v)
		}
	}

	// Default the Content-Type to application/json if not provided on reply
	if w.Header().Get("content-type") == "" {
		w.Header().Set("content-type", "application/json")
	}
	if reply.Meta.Status <= 0 {
		reply.Meta.Status = http.StatusOK
	}

	w.WriteHeader(reply.Meta.Status)

	// Optionally decode the response body
	var bodyBytes []byte
	if reply.BodyEncoding == "base64" && reply.Body != "" {
		bodyBytes, err = base64.StdEncoding.DecodeString(reply.Body)
		if err != nil {
			return err
		}
	} else {
		bodyBytes = []byte(reply.Body)
	}

	// Write the response body
	_, err = w.Write(bodyBytes)
	if err != nil || reply.Meta.Status >= 400 {
		return err
	}

	return nil
}

func (m *LambdaMiddleware) invokeLambda(ctx context.Context, req *Request) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	log := m.log.With(zap.Any("function", []string{m.FunctionName}))
	startTime := time.Now()

	resp, err := m.svc.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &m.FunctionName,
		Payload:      payload,
	})

	log = log.With(zap.Duration("duration", time.Since(startTime))).Named("exit")
	if err != nil {
		log.Error("", zap.Error(err))
		return nil, err
	}

	if resp.FunctionError != nil {
		err = fmt.Errorf("Function error: %s: %w", *resp.FunctionError, errors.New(string(resp.Payload)))
		log.Error("", zap.Error(err))
		return nil, err
	}

	log.Info("")
	return resp.Payload, nil
}

// Cleanup implements caddy.Cleanup
// TODO: ensure all running processes are terminated.
func (m *LambdaMiddleware) Cleanup() error {
	return nil
}
