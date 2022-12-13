package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sobhanatar/kauth/config"
	"io"
	"net/http"
)

const (
	PluginName = "kauth"
	CfgAdr     = "/opt/krakend/plugins/kauth.json"
	UserUuid   = "User-Uuid"
)

// ClientRegisterer is the symbol the plugin loader will try to load. It must implement the RegisterClient interface
var ClientRegisterer = registerer(PluginName)

type registerer string

var (
	logger Logger = nil
	cfg    config.KauthConfig
)

type IdentityResponse struct {
	Data struct {
		Result []struct {
			Id   string `json:"id,omitempty"`
			Uuid string `json:"uuid,omitempty"`
		} `json:"result,omitempty"`
	} `json:"data,omitempty"`
}

func (registerer) RegisterLogger(v interface{}) {
	l, ok := v.(Logger)
	if !ok {
		return
	}
	logger = l
	logger.Debug(fmt.Sprintf("[PLUGIN: %s] Logger loaded", ClientRegisterer))
}

func (r registerer) RegisterClients(f func(
	name string,
	handler func(context.Context, map[string]interface{}) (http.Handler, error),
)) {
	f(string(r), r.registerClients)
}

func (r registerer) registerClients(_ context.Context, extra map[string]interface{}) (http.Handler, error) {
	name, ok := extra["name"].(string)
	if !ok {
		return nil, errors.New("wrong config")
	}

	if name != string(r) {
		return nil, fmt.Errorf("unknown register %s", name)
	}

	// The config variable contains all the keys you have defined in the configuration:
	//_, _ = extra["kauth"].(map[string]interface{})

	if err := cfg.ParseClient(CfgAdr); err != nil {
		return nil, fmt.Errorf(err.Error())
	}

	logger.Info(fmt.Sprintf("config loaded. identity path is: %s", cfg.Path))

	// return the actual handler wrapping or your custom logic, so it can be used as a replacement for the default http handler
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		var (
			idInfo   IdentityResponse
			userUuid string
		)

		req.Header.Del(UserUuid)
		bearerToken := req.Header.Get("Authorization")
		if len(bearerToken) == 0 {
			logger.Info("the request doesn't have a bearerToken token and will be executed directly")
			err := processMainRequest(w, req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		logger.Info(fmt.Sprintf("the request has bearerToken token. getting %s from: %s", UserUuid, cfg.Path))

		authResp, err := processAuthRequest(bearerToken)
		if err != nil {
			logger.Error(fmt.Sprintf("calling auth point returned with error: %s", err.Error()))
			logger.Info("calling main endpoint without ")
		}
		defer authResp.Body.Close()

		if authResp.StatusCode == 200 {
			body, _ := io.ReadAll(authResp.Body)
			_ = json.Unmarshal(body, &idInfo)

			userUuid = idInfo.Data.Result[0].Uuid
			logger.Info(fmt.Sprintf("Executing primary request using %s: %s", UserUuid, userUuid))
			req.Header.Add(UserUuid, userUuid)
		}

		err = processMainRequest(w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Info("Removing User-Uuid from response header...")
		w.Header().Del(UserUuid)

		return
	}), nil
}

func processAuthRequest(bearer string) (idResp *http.Response, err error) {
	idReq, err := http.NewRequest(http.MethodGet, cfg.Path, nil)
	if err != nil {
		return
	}

	idReq.Header.Add("Authorization", bearer)
	idReq.Header.Add("Accept", "application/json")
	idReq.Header.Add("Content-Type", "application/json")

	idResp, err = http.DefaultClient.Do(idReq)
	return
}

func processMainRequest(w http.ResponseWriter, req *http.Request) (err error) {
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return
	}
	defer resp.Body.Close()

	err = setResponse(w, resp)
	return
}

func setResponse(w http.ResponseWriter, resp *http.Response) (err error) {
	setHeader(w, resp)
	w.WriteHeader(resp.StatusCode)
	err = setBody(w, resp)
	return
}

func setHeader(w http.ResponseWriter, resp *http.Response) {
	for k, hs := range resp.Header {
		for _, h := range hs {
			w.Header().Add(k, h)
		}
	}
}

func setBody(w http.ResponseWriter, resp *http.Response) (err error) {
	if resp.Body != nil {
		if _, err = io.Copy(w, resp.Body); err != nil {
			return
		}
	}

	return
}

func main() {}

type Logger interface {
	Debug(v ...interface{})
	Info(v ...interface{})
	Warning(v ...interface{})
	Error(v ...interface{})
	Critical(v ...interface{})
	Fatal(v ...interface{})
}
