# TO DO

# HTTP2 Server
## Questions

## Header Frames
* Requires implementing HPACK compression and decompression.
* Requires understanding how to parse headers

## Implement other frame payloads in `frame/`.

## Read about streams, dependencies and priorities.

# HPACK - header compression package

## Implement huffman coding trie

## Transcribe huffman code from the RFC

## Implement octet string encoding and decoding.
* Strings require encoding/decoding integer values
  according to the integer spec in the RFC.
* Additionally, they require the huffman code implementation
  so actors can support compressed header lines.
## Encode static header table in the package.
The RFC predefines several static header name/value pairs that
get a special index in a "static table". From my current understanding,
an HPACK header can use this special index to refer to one of these
predefined header definitions.

## What doth the dynamic table?
It seems like new header key/value pairs can be added to the
header dynamic table. What the hell is this and how does it work?
