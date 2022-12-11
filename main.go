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
	logger.Info(fmt.Sprintf("Config loaded. Identity Path is: %s", cfg.Path))

	// return the actual handler wrapping or your custom logic, so it can be used as a replacement for the default http handler
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		bearer := req.Header.Get("Authorization")
		if len(bearer) == 0 {
			logger.Info("Authorization token is: ", bearer)
			logger.Info("The request doesn't have a bearer token and will be executed directly")
			err := processMainRequest(w, req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
			}
			return
		}

		logger.Info("The request has bearer token. Getting User-Uuid from: ", cfg.Path)
		var idInfo IdentityResponse
		idInfo, err := processAuthRequest(bearer)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		userUuid := idInfo.Data.Result[0].Uuid

		logger.Info("Executing primary request using User-Uuid: %s", userUuid)
		req.Header.Add("User-Uuid", userUuid)
		err = processMainRequest(w, req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Info("Removing User-Uuid from response header...")
		w.Header().Del("User-Uuid")

		return
	}), nil
}

func processAuthRequest(bearer string) (idInfo IdentityResponse, err error) {
	idReq, err := http.NewRequest(http.MethodGet, cfg.Path, nil)
	if err != nil {
		return
	}

	idReq.Header.Add("Authorization", bearer)
	idReq.Header.Add("Accept", "application/json")
	idReq.Header.Add("Content-Type", "application/json")

	idResp, err := http.DefaultClient.Do(idReq)
	if err != nil {
		return
	}
	defer idResp.Body.Close()

	body, err := io.ReadAll(idResp.Body)
	if err = json.Unmarshal(body, &idInfo); err != nil {
		return
	}

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
