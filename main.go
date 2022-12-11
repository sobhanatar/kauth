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
		authToken := req.Header.Get("Authorization")
		logger.Warning("Authorization", authToken)

		// no auth token, resume the regular request
		if len(authToken) == 0 {
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

			w.WriteHeader(resp.StatusCode)

			if resp.Body != nil {
				if _, err = io.Copy(w, resp.Body); err != nil {
					http.Error(w, err.Error(), http.StatusInternalServerError)
					return
				}
			}
			return
		}

		idReq, err := http.NewRequest("GET", cfg.Path, nil)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		idReq.Header.Add("Authorization", authToken)
		idReq.Header.Add("Accept", "application/json")
		idReq.Header.Add("Content-Type", "application/json")

		idResp, err := http.DefaultClient.Do(idReq)
		if err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}
		defer idResp.Body.Close()

		var idInfo IdentityResponse
		body, err := io.ReadAll(idResp.Body)
		if err = json.Unmarshal(body, &idInfo); err != nil {
			http.Error(w, err.Error(), http.StatusInternalServerError)
			return
		}

		req.Header.Add("User-Uuid", idInfo.Data.Result[0].Uuid)
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
			if _, err = io.Copy(w, resp.Body); err != nil {
				http.Error(w, err.Error(), http.StatusInternalServerError)
				return
			}
		}
		return
	}), nil
}

func setBody(w http.ResponseWriter, resp *http.Response) {
}

func setHeaderAndStatus(w http.ResponseWriter, resp *http.Response) {
}

func init() {

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
