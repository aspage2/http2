
## Client
Implement a client that uses HTTP/2.

## Session Settings
Spec includes several session configurations that are negotiated during the connection.
The server must acknowledge and enforce the settings included in the spec. This 
includes things like max frame size.

## Flow Control
Implement the "Flow Control" portion of the RFC.

## Stream Priority
Implement the "stream priority" portion of the RFC.

## Continuation frames and multiple data frames
The RFC permits multiple frames to be used to pass large header or body payloads
back to the user.

## Server Examples
* A file server that can serve static sites.
* A simple HTTP api with path parsing
* Server-sent events

