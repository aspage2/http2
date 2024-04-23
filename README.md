# HTTP2 Implementation

This project implements RFC 7540 (HTTP/2) and 7541 (HPACK). It is meant to be
used for education & hacking and should not be used in production settings.

## Running the example

Requires Go 1.20 or later.

```
go run .
```

You can use any HTTP/2 client to test this server. The easiest one for me
has been curl:

```
curl -k --http2 https://localhost:8000
```

The python community built the `httpx` library which supports http2 and 
has a  "requests-like interface".

```py

# pip install httpx[http2]

import httpx

print(httpx.get("https://localhost:8000/", verify=False))
```

## Happy hacking!
