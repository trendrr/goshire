package cheshire

import (
	"crypto/rand"
	// "crypto/sha256"
	"encoding/base32"
	// "fmt"
	// "hash"
	// "log"
	"io"
	"strings"
)

// Sends an error response to the channel
func SendError(txn *Txn, code int, message string) (int, error) {
	// log.Printf("SEND ERROR: %s : %s",txn.TxnId(), message)
	resp := NewError(txn, code, message)
	c, err := txn.Write(resp)
	return c, err
}

// sends a standard 200 success response.  
// will close the current transaction 
func SendSuccess(txn *Txn) {
	// log.Printf("SEND SUCCESS: %s", txn.TxnId())
	txn.Write(NewResponse(txn))
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
