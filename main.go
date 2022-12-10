package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
)

const PluginName = "kauth"

const identityPath = "http://172.17.0.2:80/api/identity"

// ClientRegisterer is the symbol the plugin loader will try to load. It must implement the RegisterClient interface
var ClientRegisterer = registerer(PluginName)

type registerer string

var logger Logger = nil

type Response struct {
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
	// check the passed configuration and initialize the plugin
	name, ok := extra["name"].(string)
	if !ok {
		return nil, errors.New("wrong config")
	}

	if name != string(r) {
		return nil, fmt.Errorf("unknown register %s", name)
	}

	// The config variable contains all the keys you have defined in the configuration:
	//_, _ = extra["kauth"].(map[string]interface{})

	// return the actual handler wrapping or your custom logic, so it can be used as a replacement for the default http handler
	return http.HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		authToken := req.Header.Get("Authorization")

		// no auth token, resume the regular request
		if len(authToken) == 0 {
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}

			for k, hs := range resp.Header {
				for _, h := range hs {
					w.Header().Add(k, h)
				}
			}

			w.WriteHeader(resp.StatusCode)

			if resp.Body != nil {
				_, _ = io.Copy(w, resp.Body)
				_ = resp.Body.Close()
			}
			return
		}

		idReq, _ := http.NewRequest("GET", identityPath, nil)
		idReq.Header.Add("Authorization", authToken)

		idResp, err := http.DefaultClient.Do(idReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer idResp.Body.Close()

		body, err := io.ReadAll(idResp.Body)
		var userInfo Response
		if err = json.Unmarshal(body, &userInfo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		req.Header.Add("User-Uuid", userInfo.Data.Result[0].Uuid)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		for k, hs := range resp.Header {
			for _, h := range hs {
				w.Header().Add(k, h)
			}
		}

		w.Header().Del("User-Uuid")

		w.WriteHeader(resp.StatusCode)

		if resp.Body != nil {
			_, _ = io.Copy(w, resp.Body)
			defer resp.Body.Close()
		}
		return
	}), nil
}

func processWithoutAuth(w http.ResponseWriter, req *http.Request) {
}

func setBody(w http.ResponseWriter, resp *http.Response) {
}

func setHeaderAndStatus(w http.ResponseWriter, resp *http.Response) {
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
