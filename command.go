package caddyawslambda

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/aws/aws-sdk-go-v2/config"
	"github.com/aws/aws-sdk-go-v2/service/lambda"
	"github.com/caddyserver/caddy/v2"
	"github.com/pkg/errors"
	"go.uber.org/zap"
)

// Cmd is the module configuration
type Cmd struct {
	// The command to run.
	Function string `json:"function,omitempty"`

	// Timeout for the command. The command will be killed
	// after timeout has elapsed if it is still running.
	// Defaults to 10s.
	Timeout string `json:"timeout,omitempty"`

	timeout time.Duration // ease of use after parsing timeout string
	log     *zap.Logger

	svc *lambda.Client
}

// Provision implements caddy.Provisioner.
func (m *Cmd) provision(ctx caddy.Context, cm caddy.Module) error {
	m.log = ctx.Logger(cm)

	// timeout
	if m.Timeout == "" {
		m.Timeout = "10s"
	}
	dur, err := time.ParseDuration(m.Timeout)
	if err != nil {
		return err
	}
	m.timeout = dur

	cfg, err := config.LoadDefaultConfig(ctx)
	if err != nil {
		return fmt.Errorf("unable to load SDK config, %w", err)
	}

	m.svc = lambda.NewFromConfig(cfg)
	return nil
}

// Validate implements caddy.Validator.
func (m Cmd) validate() error {
	if m.Function == "" {
		return fmt.Errorf("function is required")
	}
	return nil
}

func (m *Cmd) invokeLambda(ctx context.Context, req *Request) ([]byte, error) {
	ctx, cancel := context.WithTimeout(ctx, m.timeout)
	defer cancel()

	payload, err := json.Marshal(req)
	if err != nil {
		return nil, err
	}

	log := m.log.With(zap.Any("function", []string{m.Function}))
	startTime := time.Now()

	resp, err := m.svc.Invoke(ctx, &lambda.InvokeInput{
		FunctionName: &m.Function,
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
