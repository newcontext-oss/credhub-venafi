// Copyright 2019 New Context, Inc.
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//     http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package main

import (
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path/filepath"

	yaml "gopkg.in/yaml.v2"
)

const CVLogFilename string = "cv.log"

var VERBOSE int32 = 4
var INFO int32 = 3
var STATUS int32 = 2
var ERROR int32 = 1

var logLevel int32 = STATUS

type YAMLConfig struct {
	VcertUsername   string `yaml:"vcert_username"`
	VcertPassword   string `yaml:"vcert_password"`
	VcertZone       string `yaml:"vcert_zone"`
	VcertBaseUrl    string `yaml:"vcert_base_url"`
	ConnectorType   string `yaml:"connector_type"`
	ClientID        string `yaml:"credhub_client_id"`
	ClientSecret    string `yaml:"credhub_client_secret"`
	CredhubUsername string `yaml:"credhub_username"`
	CredhubPassword string `yaml:"credhub_password"`
	CredhubEndpoint string `yaml:"credhub_endpoint"`
	LogLevel        string `yaml:"log_level"`

	SkipTLSValidation bool `yaml:"skip_tls_validation"`
}

func readConfigFile(homedir string, path string) (*YAMLConfig, error) {
	configpath := filepath.Join(homedir, path)
	tt := YAMLConfig{}
	file, err := ioutil.ReadFile(configpath)
	if err != nil {
		return nil, err
	}

	err = yaml.Unmarshal(file, &tt)
	if err != nil {
		return nil, err
	}
	if tt.ConnectorType == "" {
		tt.ConnectorType = "tpp"
	}

	switch tt.LogLevel {
	case "error":
		logLevel = ERROR
	case "info":
		logLevel = INFO
	case "verbose":
		logLevel = VERBOSE
	case "status":
		logLevel = STATUS
	}

	logfilePath := CVLogFilename
	f, err := os.OpenFile(logfilePath, os.O_RDWR|os.O_CREATE|os.O_APPEND, 0666)
	if err != nil {
		return nil, fmt.Errorf("error opening log file: %v", err)
	}

	log.SetOutput(f)
	return &tt, nil
}
