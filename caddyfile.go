package caddyawslambda

import (
	"github.com/caddyserver/caddy/v2/caddyconfig/caddyfile"
	"github.com/caddyserver/caddy/v2/caddyconfig/httpcaddyfile"
	"github.com/caddyserver/caddy/v2/modules/caddyhttp"
)

func init() {
	httpcaddyfile.RegisterHandlerDirective("awslambda", parseCaddyfile)
}

func parseCaddyfile(h httpcaddyfile.Helper) (caddyhttp.MiddlewareHandler, error) {
	m := &LambdaMiddleware{}
	err := m.UnmarshalCaddyfile(h.Dispenser)
	if err != nil {
		return nil, err
	}
	return m, nil
}

// UnmarshalCaddyfile configures the global directive from Caddyfile.
// Syntax:
//
//   awslambda [<matcher>] {
//       function <function name>
//       timeout  <duration>
//   }
func (m *LambdaMiddleware) UnmarshalCaddyfile(d *caddyfile.Dispenser) error {
	for d.Next() {
		if d.NextArg() {
			return d.ArgErr()
		}
		for d.NextBlock(0) {
			switch d.Val() {
			case "function":
				if m.FunctionName != "" {
					return d.Err("function already specified")
				}
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.FunctionName = d.Val()
			case "timeout":
				if !d.NextArg() {
					return d.ArgErr()
				}
				m.Timeout = d.Val()
			default:
				return d.Errf("unrecognized subdirective: %s", d.Val())
			}
		}
	}
	return nil
}
