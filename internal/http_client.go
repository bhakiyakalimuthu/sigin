/*
Copyright © 2022
Author Bhakiyaraj Kalimuthu
Email bhakiya.kalimuthu@gmail.com
*/

package internal

import (
	"encoding/json"
	"errors"
	"go.uber.org/zap"
	"io"
	"net/http"
	"sigin/store"
	"time"
)

type HttpClient interface {
	GetMethodSignature(pageURL string) (*Response, error)
}

type httpClient struct {
	logger     *zap.Logger
	httpClient *http.Client
	store      store.Store
}

func NewHttpClient(logger *zap.Logger, store store.Store) HttpClient {
	client := &http.Client{Timeout: time.Second * 10} // default timeout set to 5s
	return &httpClient{
		logger:     logger,
		httpClient: client,
		store:      store,
	}
}

//GetMethodSignature http client to make http post request
func (n *httpClient) GetMethodSignature(pageURL string) (*Response, error) {
	n.logger.Debug("making http request")
	// create http request
	req, err := http.NewRequest(http.MethodGet, pageURL, nil)
	if err != nil {
		n.logger.Error("failed to make new request", zap.Error(err))
		return nil, err
	}
	// make post request
	res, err := n.httpClient.Do(req)
	if err != nil {
		n.logger.Error("failed to make new request", zap.Error(err))
		return nil, err
	}

	bodyBytes, err := io.ReadAll(res.Body)
	if err != nil {
		n.logger.Error("failed to read body", zap.Error(err))
		return nil, err
	}
	defer res.Body.Close()
	var out *Response
	if err = json.Unmarshal(bodyBytes, &out); err != nil {
		n.logger.Error("failed to unmarshal body bytes", zap.Error(err))
		return nil, err
	}
	if out == nil {
		return nil, errors.New("empty repose")
	}
	n.logger.Info("successfully received page info", zap.String("url", out.Next), zap.Int("jobCount", len(out.Results)))
	return out, nil
}
