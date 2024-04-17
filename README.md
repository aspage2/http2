# HTTP/2 Project

Project to implement RFC 7540 & RFC 7541 (HTTP/2).

## Notes

The server is set up for tls using a self-signed certificate.
It will negotiate for HTTP/2.0 in the ALPN layer during the TLS
handshake. **As a future task, this server should probably support
upgrading to HTTP/2 from an HTTP/1.1 connection, as well as h2c
(http2 cleartext).

**This is a toy project for learning & hacking!**. It should not
be used in a proudction setting.

I'm keeping a scratchlist of TODO tasks in TODO.md.

## Testing the Server
Run the server with `go run .`. All it does right now is log
frames that the client sends to it from

In another window, use an http2 client to make a server request.

```
curl -k --http2 https://localhost:8000
```
