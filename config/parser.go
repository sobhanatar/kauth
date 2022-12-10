package config

import (
	"encoding/json"
	"errors"
	"fmt"
	"github.com/sobhanatar/kauth/messages"
	"os"
)

func (cfg *KauthConfig) ParseClient(addr string) (err error) {
	f, err := os.ReadFile(addr)
	if err != nil {
		return errors.New(fmt.Sprintf(messages.ClientConfigFileError, err.Error()))
	}

	if err = json.Unmarshal(f, cfg); err != nil {
		return errors.New(fmt.Sprintf(messages.ClientConfigFIleUnmarshalError, err.Error()))
	}

	return
}
