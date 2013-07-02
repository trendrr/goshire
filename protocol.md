# Protocol Documentation


### Binary Protocol

The binary protocol is similar to the json protocol, just a different encoding, meant to be faster to encode and decode then json for high performance critical applications (databases, queues, ect). Please read the json protocol spec for more details, as this doc simply describes the encoding.

Request Format

* All integers are signed big endian.
* Variable length fields (arrays and strings) are all prefixed by a 'length' int
* All length fields are int16, unless otherwise noted
* All strings are utf8 encoded

### Hello Packet

Clients should send this packet on initial connection.  No response will be sent from the server.  Server will disconnect without a proper hello

```
[length][version(string)][space][useragent(string)]
```

After hello is sent, then the client is free to start sending requests, and recieving responses.
The protocol is async (same as the json protocol), so clients should always listen for new responses.  There are only 2 packet types: REQUEST and RESPONSE.  


### REQUEST

```
[header_length (int32)]
[partition (int16)]
[length][shard key (string)]
[length][txn_id (string)]
[txn_accept(int8)]
[method(int8)]
[length][uri (string)]
[param_encoding (int8)]
[params (array)]
[content_encoding (int8)]
[content_length (int32)][content (array)]
```
 -- Note the shard key and partition sections are only used for goshire-shard router requests. Other requests should zero those fields


### RESPONSE

```
[header_length (int32)]
[length][txn_id (string)]
[txn_status(int8)]
[status (int16)]
[length][status_message (string)]
[content_encoding (int8)]
[content_length (int32)][content (array)]
```
