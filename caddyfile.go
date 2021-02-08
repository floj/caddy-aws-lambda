package caddyawslambda

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
)

// parseHandlerCaddyfile unmarshals tokens from h into a new Middleware.
func parseHandlerCaddyfile(h httpcaddyfile.Helper) ([]httpcaddyfile.ConfigValue, error) {
	if !h.Next() {
		return nil, h.ArgErr()
	}
	var c Cmd

	// logic copied from RegisterHandlerDirective to customize.
	matcherSet, ok, err := h.MatcherToken()
	if err != nil {
		return nil, err
	}
	if ok {
		h.Dispenser.Delete()
	}
	h.Dispenser.Reset()

	// parse Caddyfile
	err = c.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}

	m := Middleware{Cmd: c}
	return h.NewRoute(matcherSet, m), nil
}

// UnmarshalCaddyfile configures the global directive from Caddyfile.
// Syntax:
//
//   awslambda [<matcher>] [<function name>] {
//       function <text>
//       timeout  <duration>
//   }
//
func (m *Cmd) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	// consume "awslambda", then grab the command, if present.
	if d.NextArg() && d.NextArg() {
		m.Function = d.Val()
	}

	// parse the next block
	return m.unmarshalBlock(d)
}

func (m *Cmd) unmarshalBlock(d *caddyfile.Dispenser) error {
	for d.NextBlock(0) {
		switch d.Val() {
		case "function":
			if m.Function != "" {
				return d.Err("function specified twice")
			}
			if !d.Args(&m.Function) {
				return d.ArgErr()
			}
		case "timeout":
			if !d.Args(&m.Timeout) {
				return d.ArgErr()
			}
		}
	}

	return nil
}
