## UI Client interface

These data types are defined in the channel between clef and the UI
### SignDataRequest

SignDataRequest contains information about a pending request to sign some data. The data to be signed can be of various types, defined by content-type. Clef has done most of the work in canonicalizing and making sense of the data, and it's up to the UI to present the user with the contents of the `message`

Example:
```json
{
  "content_type": "text/plain",
  "address": "gdDEADbEeF000000000000000000000000DeaDbeEf",
  "raw_data": "GUV0aGVyZXVtIFNpZ25lZCBNZXNzYWdlOgoxMWhlbGxvIHdvcmxk",
  "messages": [
    {
      "name": "message",
      "value": "\u0019Gdtu Signed Message:\n11hello world",
      "type": "text/plain"
    }
  ],
  "hash": "gdd9eba16ed0ecae432b71fe008c98cc872bb4cc214d3220a36f365326cf807d68",
  "meta": {
    "remote": "localhost:9999",
    "local": "localhost:8545",
    "scheme": "http",
    "User-Agent": "Firefox 3.2",
    "Origin": "www.malicious.ru"
  }
}
```
### SignDataResponse - approve

Response to SignDataRequest

Example:
```json
{
  "approved": true
}
```
### SignDataResponse - deny

Response to SignDataRequest

Example:
```json
{
  "approved": false
}
```
### SignTxRequest

SignTxRequest contains information about a pending request to sign a transaction. Aside from the transaction itself, there is also a `call_info`-struct. That struct contains messages of various types, that the user should be informed of.

As in any request, it's important to consider that the `meta` info also contains untrusted data.

The `transaction` (on input into clef) can have either `data` or `input` -- if both are set, they must be identical, otherwise an error is generated. However, Clef will always use `data` when passing this struct on (if Clef does otherwise, please file a ticket)

Example:
```json
{
  "transaction": {
    "from": "gdDEADbEeF000000000000000000000000DeaDbeEf",
    "to": null,
    "gas": "gd3e8",
    "gasPrice": "gd5",
    "value": "gd6",
    "nonce": "gd1",
    "data": "gd01020304"
  },
  "call_info": [
    {
      "type": "Warning",
      "message": "Somgdtuing looks odd, show this message as a warning"
    },
    {
      "type": "Info",
      "message": "User should see this aswell"
    }
  ],
  "meta": {
    "remote": "localhost:9999",
    "local": "localhost:8545",
    "scheme": "http",
    "User-Agent": "Firefox 3.2",
    "Origin": "www.malicious.ru"
  }
}
```
### SignTxResponse - approve

Response to request to sign a transaction. This response needs to contain the `transaction`, because the UI is free to make modifications to the transaction.

Example:
```json
{
  "transaction": {
    "from": "gdDEADbEeF000000000000000000000000DeaDbeEf",
    "to": null,
    "gas": "gd3e8",
    "gasPrice": "gd5",
    "value": "gd6",
    "nonce": "gd4",
    "data": "gd04030201"
  },
  "approved": true
}
```
### SignTxResponse - deny

Response to SignTxRequest. When denying a request, there's no need to provide the transaction in return

Example:
```json
{
  "transaction": {
    "from": "gd",
    "to": null,
    "gas": "gd0",
    "gasPrice": "gd0",
    "value": "gd0",
    "nonce": "gd0",
    "data": null
  },
  "approved": false
}
```
### OnApproved - SignTransactionResult

SignTransactionResult is used in the call `clef` -> `OnApprovedTx(result)`

This occurs _after_ successful completion of the entire signing procedure, but right before the signed transaction is passed to the external caller. This Method (and data) can be used by the UI to signal to the user that the transaction was signed, but it is primarily useful for ruleset implementations.

A ruleset that implements a rate limitation needs to know what transactions are sent out to the external interface. By hooking into this Methods, the ruleset can maintain track of that count.

**OBS:** Note that if an attacker can restore your `clef` data to a previous point in time (e.g through a backup), the attacker can reset such windows, even if he/she is unable to decrypt the content.

The `OnApproved` Method cannot be responded to, it's purely informative

Example:
```json
{
  "raw": "gdf85d640101948a8eafb1cf62bfbeb1741769dae1a9dd47996192018026a0716bd90515acb1e68e5ac5867aa11a1e65399c3349d479f5fb698554ebc6f293a04e8a4ebfff434e971e0ef12c5bf3a881b06fd04fc3f8b8a7291fb67a26a1d4ed",
  "tx": {
    "nonce": "gd64",
    "gasPrice": "gd1",
    "gas": "gd1",
    "to": "gd8a8eafb1cf62bfbeb1741769dae1a9dd47996192",
    "value": "gd1",
    "input": "gd",
    "v": "gd26",
    "r": "gd716bd90515acb1e68e5ac5867aa11a1e65399c3349d479f5fb698554ebc6f293",
    "s": "gd4e8a4ebfff434e971e0ef12c5bf3a881b06fd04fc3f8b8a7291fb67a26a1d4ed",
    "hash": "gd662f6d772692dd692f1b5e8baa77a9ff95bbd909362df3fc3d301aafebde5441"
  }
}
```
### UserInputRequest

Sent when clef needs the user to provide data. If 'password' is true, the input field should be treated accordingly (echo-free)

Example:
```json
{
  "prompt": "The question to ask the user",
  "title": "The title here",
  "isPassword": true
}
```
### UserInputResponse

Response to UserInputRequest

Example:
```json
{
  "text": "The textual response from user"
}
```
### ListRequest

Sent when a request has been made to list addresses. The UI is provided with the full `account`s, including local directory names. Note: this information is not passed back to the external caller, who only sees the `address`es.

Example:
```json
{
  "accounts": [
    {
      "address": "gddeadbeef000000000000000000000000deadbeef",
      "url": "keystore:///path/to/keyfile/a"
    },
    {
      "address": "gd1111111122222222222233333333334444444444",
      "url": "keystore:///path/to/keyfile/b"
    }
  ],
  "meta": {
    "remote": "localhost:9999",
    "local": "localhost:8545",
    "scheme": "http",
    "User-Agent": "Firefox 3.2",
    "Origin": "www.malicious.ru"
  }
}
```
### ListResponse

Response to list request. The response contains a list of all addresses to show to the caller. Note: the UI is free to respond with any address the caller, regardless of whether it exists or not

Example:
```json
{
  "accounts": [
    {
      "address": "gd0000000000000000000000000000000000000000",
      "url": ".. ignored .."
    },
    {
      "address": "gdffffffffffffffffffffffffffffffffffffffff",
      "url": ""
    }
  ]
}
```
