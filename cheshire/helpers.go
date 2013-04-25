package cheshire

import (
	"crypto/rand"
	// "crypto/sha256"
	"encoding/base32"
	// "fmt"
	// "hash"
	"io"
	"strings"
)

// Sends an error response to the channel
func SendError(txn *Txn, code int, message string) (int, error) {
	resp := NewError(txn, code, message)
	c, err := txn.Write(resp)
	return c, err
}

// Creates a new response based on this request txn.
// auto fills the txn id
func NewResponse(txn RequestTxnId) *Response {
	response := newResponse()
	response.SetTxnId(txn.TxnId())
	return response
}

func NewError(txn RequestTxnId, code int, message string) *Response {
	response := NewResponse(txn)
	response.SetStatus(code, message)
	return response
}

func RandString(length int) string {
	k := make([]byte, length*2)
	if _, err := io.ReadFull(rand.Reader, k); err != nil {
		return ""
	}
	str := strings.TrimRight(
		base32.StdEncoding.EncodeToString(k), "=")
	return str[:length]
}
