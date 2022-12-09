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

const identityPath = "http://172.17.0.5:80/api/identity"

// ClientRegisterer is the symbol the plugin loader will try to load. It must implement the RegisterClient interface
var ClientRegisterer = registerer(PluginName)

type registerer string

var logger Logger = nil

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
	config, _ := extra["kauth"].(map[string]interface{})

	// The plugin will look for this path:
	path, _ := config["path"].(string)
	logger.Debug(fmt.Sprintf("The plugin is now hijacking the path %s", path))

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
				io.Copy(w, resp.Body)
				resp.Body.Close()
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

		var userInfo map[string]interface{}
		if err = json.NewDecoder(idResp.Body).Decode(&userInfo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		logger.Debug("user info data", userInfo)

		req.Header.Add("User-Uuid", userInfo["uuid"].(string))
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer resp.Body.Close()

		logger.Debug("foo header", resp.Header.Get("Foo"))
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
