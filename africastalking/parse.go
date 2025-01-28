package at

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"strings"

	"git.grassecon.net/grassrootseconomics/common/phone"
	"git.grassecon.net/grassrootseconomics/visedriver/errors"
)

type ATRequestParser struct {
	Context context.Context
}

func (arp *ATRequestParser) GetSessionId(ctx context.Context, rq any) (string, error) {
	rqv, ok := rq.(*http.Request)
	if !ok {
		logg.Warnf("got an invalid request", "req", rq)
		return "", errors.ErrInvalidRequest
	}

	// Capture body (if any) for logging
	body, err := io.ReadAll(rqv.Body)
	if err != nil {
		logg.Warnf("failed to read request body", "err", err)
		return "", fmt.Errorf("failed to read request body: %v", err)
	}
	// Reset the body for further reading
	rqv.Body = io.NopCloser(bytes.NewReader(body))

	// Log the body as JSON
	bodyLog := map[string]string{"body": string(body)}
	logBytes, err := json.Marshal(bodyLog)
	if err != nil {
		logg.Warnf("failed to marshal request body", "err", err)
	} else {
		decodedStr := string(logBytes)
		sessionId, err := extractATSessionId(decodedStr)
		if err != nil {
			context.WithValue(arp.Context, "at-session-id", sessionId)
		}
		logg.Debugf("Received request:", decodedStr)
	}

	if err := rqv.ParseForm(); err != nil {
		logg.Warnf("failed to parse form data", "err", err)
		return "", fmt.Errorf("failed to parse form data: %v", err)
	}

	phoneNumber := rqv.FormValue("phoneNumber")
	if phoneNumber == "" {
		return "", fmt.Errorf("no phone number found")
	}

	formattedNumber, err := phone.FormatPhoneNumber(phoneNumber)
	if err != nil {
		logg.Warnf("failed to format phone number", "err", err)
		return "", fmt.Errorf("failed to format number")
	}

	return formattedNumber, nil
}

func (arp *ATRequestParser) GetInput(rq any) ([]byte, error) {
	rqv, ok := rq.(*http.Request)
	if !ok {
		return nil, errors.ErrInvalidRequest
	}
	if err := rqv.ParseForm(); err != nil {
		return nil, fmt.Errorf("failed to parse form data: %v", err)
	}

	text := rqv.FormValue("text")

	parts := strings.Split(text, "*")
	if len(parts) == 0 {
		return nil, fmt.Errorf("no input found")
	}

	trimmedInput := strings.TrimSpace(parts[len(parts)-1])
	return []byte(trimmedInput), nil
}

func parseQueryParams(query string) map[string]string {
	params := make(map[string]string)

	queryParams := strings.Split(query, "&")
	for _, param := range queryParams {
		// Split each key-value pair by '='
		parts := strings.SplitN(param, "=", 2)
		if len(parts) == 2 {
			params[parts[0]] = parts[1]
		}
	}
	return params
}

func extractATSessionId(decodedStr string) (string, error) {
	var data map[string]string
	err := json.Unmarshal([]byte(decodedStr), &data)

	if err != nil {
		logg.Errorf("Error unmarshalling JSON: %v", err)
		return "", nil
	}
	decodedBody, err := url.QueryUnescape(data["body"])
	if err != nil {
		logg.Errorf("Error URL-decoding body: %v", err)
		return "", nil
	}
	params := parseQueryParams(decodedBody)

	sessionId := params["sessionId"]
	return sessionId, nil

}
