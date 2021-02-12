// Copyright (c) 2019 Snowflake Computing Inc. All right reserved.

package gosnowflake

import (
	"context"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"net/http"
	"net/url"
	"time"

	"github.com/google/uuid"
)

const (
	// One hour interval should be good enough to renew tokens for four hours master token validity
	heartBeatInterval = 3600 * time.Second
)

type heartbeat struct {
	restful      *snowflakeRestful
	shutdownChan chan bool
}

func (hc *heartbeat) run() {
	hbTicker := time.NewTicker(heartBeatInterval)
	defer hbTicker.Stop()
	for {
		select {
		case <-hbTicker.C:
			err := hc.heartbeatMain()
			if err != nil {
				logger.Error("failed to heartbeat")
			}
		case <-hc.shutdownChan:
			logger.Info("stopping heartbeat")
			return
		}
	}
}

func (hc *heartbeat) start() {
	hc.shutdownChan = make(chan bool)
	go hc.run()
	logger.Info("heartbeat started")
}

func (hc *heartbeat) stop() {
	hc.shutdownChan <- true
	close(hc.shutdownChan)
	logger.Info("heartbeat stopped")
}

func (hc *heartbeat) heartbeatMain() error {
	logger.Info("Heartbeating!")
	params := &url.Values{}
	params.Add(requestIDKey, uuid.New().String())
	params.Add(requestGUIDKey, uuid.New().String())
	headers := make(map[string]string)
	headers["Content-Type"] = headerContentTypeApplicationJSON
	headers["accept"] = headerAcceptTypeApplicationSnowflake
	headers["User-Agent"] = userAgent
	headers[headerAuthorizationKey] = fmt.Sprintf(headerSnowflakeToken, hc.restful.Token)

	fullURL := hc.restful.getFullURL(heartBeatPath, params)
	timeout := hc.restful.RequestTimeout
	resp, err := hc.restful.FuncPost(context.Background(), hc.restful, fullURL, headers, nil, timeout, false)
	if err != nil {
		return err
	}
	defer resp.Body.Close()
	if resp.StatusCode == http.StatusOK {
		logger.Infof("heartbeatMain: resp: %v", resp)
		var respd execResponse
		err = json.NewDecoder(resp.Body).Decode(&respd)
		if err != nil {
			logger.Infof("failed to decode JSON. err: %v", err)
			return err
		}
		if respd.Code == sessionExpiredCode {
			err = hc.restful.FuncRenewSession(context.TODO(), hc.restful, timeout)
			if err != nil {
				return err
			}
		}
		return nil
	}
	b, err := ioutil.ReadAll(resp.Body)
	if err != nil {
		logger.Errorf("failed to extract HTTP response body. err: %v", err)
		return err
	}
	logger.Infof("HTTP: %v, URL: %v, Body: %v", resp.StatusCode, fullURL, b)
	logger.Infof("Header: %v", resp.Header)
	return &SnowflakeError{
		Number:   ErrFailedToHeartbeat,
		SQLState: SQLStateConnectionFailure,
		Message:  "Failed to heartbeat.",
	}
}
