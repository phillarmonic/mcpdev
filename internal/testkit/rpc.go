package testkit

import (
	"bufio"
	"bytes"
	jsonv1 "encoding/json"
	"encoding/json/v2"
	"fmt"
	"io"
)

const jsonRPCVersion = "2.0"

// Request is a JSON-RPC 2.0 request or notification.
type Request struct {
	JSONRPC string            `json:"jsonrpc"`
	ID      any               `json:"id,omitzero"`
	Method  string            `json:"method"`
	Params  jsonv1.RawMessage `json:"params,omitzero"`
}

// Response is a JSON-RPC 2.0 response.
type Response struct {
	ID      any    `json:"id,omitzero"`
	Result  any    `json:"result,omitzero"`
	Error   *Error `json:"error,omitzero"`
	JSONRPC string `json:"jsonrpc"`
}

// Error is a JSON-RPC error object.
type Error struct {
	Data    any    `json:"data,omitzero"`
	Message string `json:"message"`
	Code    int    `json:"code"`
}

func (e *Error) Error() string {
	if e == nil {
		return ""
	}
	return fmt.Sprintf("json-rpc %d: %s", e.Code, e.Message)
}

const (
	codeParse          = -32700
	codeInvalidRequest = -32600
	codeMethodNotFound = -32601
	codeInvalidParams  = -32602
	codeHeaderMismatch = -32020
	codeMissingCap     = -32021
	codeUnsupportedVer = -32022
)

func rpcError(id any, code int, message string, data any) Response {
	return Response{
		JSONRPC: jsonRPCVersion,
		ID:      id,
		Error:   &Error{Code: code, Message: message, Data: data},
	}
}

func rpcResult(id any, result any) Response {
	return Response{JSONRPC: jsonRPCVersion, ID: id, Result: result}
}

func writeJSONLine(w io.Writer, v any) error {
	b, err := json.Marshal(v, json.Deterministic(true))
	if err != nil {
		return err
	}
	_, err = w.Write(append(b, '\n'))
	return err
}

func readJSONLine(sc *bufio.Scanner, dest any) error {
	if !sc.Scan() {
		if err := sc.Err(); err != nil {
			return err
		}
		return io.EOF
	}
	line := bytes.TrimSpace(sc.Bytes())
	if len(line) == 0 {
		return readJSONLine(sc, dest)
	}
	return json.Unmarshal(line, dest)
}
