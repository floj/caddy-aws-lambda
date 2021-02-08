# caddy-exec

Caddy v2 module for dispatching requests to AWS Lambda.

This is a port of https://github.com/coopernurse/caddy-awslambda with less features but for Caddy 2.

## Installation

```
xcaddy build \
    --with github.com/floj/caddy-awslambda
```

## Usage

```
{
  order awslambda before file_server
}

http://localhost:8080 {
  log {
    output stderr
  }
  awslambda /services/* {
    function ForwardToSlack
  }
}
```

## License

Apache 2
