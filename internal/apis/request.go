package apis

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"go.mattglei.ch/timber"
)

var IgnoreError = errors.New("Warning error when trying to make request. Ignore error.")

// sends a given http.Request and will unmarshal the JSON from the response body and return that as the given type.
func SendRequest[T any](client *http.Client, req *http.Request) (T, error) {
	ctx, cancel := context.WithTimeout(req.Context(), 1*time.Minute)
	defer cancel()
	req = req.WithContext(ctx)

	var zeroValue T // to be used as "nil" when returning errors
	resp, err := client.Do(req)
	if err != nil {
		if errors.Is(err, context.DeadlineExceeded) {
			timber.Warning("request timed out for", req.URL.String())
			return zeroValue, IgnoreError
		}
		if errors.Is(err, io.ErrUnexpectedEOF) {
			timber.Warning("unexpected EOF from", req.URL.String())
			return zeroValue, IgnoreError
		}
		if strings.Contains(err.Error(), "read: connection reset by peer") {
			timber.Warning("tcp connection reset by peer from", req.URL.String())
			return zeroValue, IgnoreError
		}
		return zeroValue, fmt.Errorf("%w sending request failed", err)
	}
	defer resp.Body.Close()

	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return zeroValue, fmt.Errorf("%w reading response body failed", err)
	}
	if resp.StatusCode != http.StatusOK {
		timber.Warning(resp.StatusCode, "returned from", req.URL.String())
		return zeroValue, IgnoreError
	}

	var data T
	err = json.Unmarshal(body, &data)
	if err != nil {
		timber.Debug(string(body))
		return zeroValue, fmt.Errorf("%w failed to parse json", err)
	}

	return data, nil
}
