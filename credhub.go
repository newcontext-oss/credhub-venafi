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
	"crypto/sha1"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"path/filepath"
	"strings"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/auth"
	"code.cloudfoundry.org/credhub-cli/credhub/auth/uaa"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/generate"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/values"
)

// ConfigLoader has configuration location info and methods to load the config
type ConfigLoader struct {
	userHomeDir    string
	cvConfigDir    string
	configFilename string
}

func (c *ConfigLoader) ensureDirExists() error {
	homedir := filepath.Join(c.userHomeDir, c.cvConfigDir)
	if _, err := os.Stat(homedir); os.IsNotExist(err) {
		err := os.Mkdir(homedir, os.ModePerm)
		if err != nil {
			return err
		}
	}

	return nil
}

func (c *ConfigLoader) writeConfig(cvConfig *CVConfig) error {
	homedir := filepath.Join(c.userHomeDir, c.cvConfigDir)

	b, err := json.Marshal(&cvConfig)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(homedir, c.configFilename), b, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (c *ConfigLoader) readConfig() (*CVConfig, error) {
	configdir := filepath.Join(c.userHomeDir, c.cvConfigDir)
	if _, err := os.Stat(configdir); os.IsNotExist(err) {
		return nil, fmt.Errorf("No config dir %s %s", configdir, err)
	}

	cvConfig := CVConfig{}
	file, err := ioutil.ReadFile(path.Join(configdir, c.configFilename))
	if err != nil {
		return nil, err
	}

	err = json.Unmarshal([]byte(file), &cvConfig)
	if err != nil {
		return nil, err
	}
	return &cvConfig, nil
}

// CVConfig contains app config info and yaml tags
type CVConfig struct {
	AccessToken       string `json:"access_token"`
	RefreshToken      string `json:"refresh_token"`
	CredhubBaseURL    string `json:"credhub_url"`
	AuthURL           string `json:"auth_url"`
	SkipTLSValidation bool   `json:"skip_tls_validation"`
}

// ICredhubProxy defines the interface for the proxy to communicate with Credhub
type ICredhubProxy interface {
	generateCertificate(name string, parameters generate.Certificate, overwrite credhub.Mode) (credentials.Certificate, error)
	putCertificate(certName string, ca string, certificate string, privateKey string) error
	deleteCert(name string) error
	list() ([]credentials.CertificateMetadata, error)
	getCertificate(name string) (credentials.Certificate, error)
}

// CredhubProxy contains the config information for the Credhub request proxy
type CredhubProxy struct {
	baseURL           string
	username          string
	password          string
	clientID          string
	clientSecret      string
	accessToken       string
	refreshToken      string
	authURL           string
	client            *credhub.CredHub
	configPath        string
	skipTLSValidation bool
}

func (cp *CredhubProxy) generateCertificate(name string, parameters generate.Certificate, overwrite credhub.Mode) (credentials.Certificate, error) {
	newCert, err := cp.client.GenerateCertificate(name, parameters, overwrite)
	verbose("newCert %+v", newCert)
	return newCert, err
}

func (cp *CredhubProxy) putCertificate(certName string, ca string, certificate string, privateKey string) error {
	c := values.Certificate{}
	c.Ca = ca
	c.Certificate = certificate
	c.PrivateKey = privateKey
	newCert, err := cp.client.SetCertificate(certName, c)
	_ = newCert
	if err != nil {
		return nil
	}
	return nil
}

func (cp *CredhubProxy) deleteCert(name string) error {
	return cp.client.Delete(name)
}

func (cp *CredhubProxy) list() ([]credentials.CertificateMetadata, error) {
	certs, err := cp.client.GetAllCertificatesMetadata()
	if err != nil {
		return nil, err
	}

	return certs, nil
}

func (cp *CredhubProxy) getCertificate(name string) (credentials.Certificate, error) {
	cred, err := cp.client.GetLatestCertificate(name)
	if err != nil {
		return credentials.Certificate{}, err
	}
	return cred, nil
}

func getThumbprint(cert string) ([sha1.Size]byte, error) {
	certStr := strings.ReplaceAll(cert, "-----BEGIN CERTIFICATE-----", "")
	certStr = strings.ReplaceAll(certStr, "-----END CERTIFICATE-----", "")
	certStr = strings.ReplaceAll(certStr, "\n", "")

	data, err := base64.StdEncoding.DecodeString(certStr)
	if err != nil {
		return [20]byte{}, err
	}

	return sha1.Sum(data), nil
}

func (cp *CredhubProxy) writeConfig(configPath string, config *CVConfig) error {
	// ensure the .cv directory is created
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	homedir := filepath.Join(home, cp.configPath)
	if _, err := os.Stat(homedir); os.IsNotExist(err) {
		os.Mkdir(homedir, os.ModePerm)
	}

	if err != nil {
		return err
	}

	b, err := json.Marshal(&config)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(homedir, "config.json"), b, os.ModePerm)
	if err != nil {
		return err
	}
	return nil
}

func (cp *CredhubProxy) authExisting() error {
	var err error
	cp.client, err = credhub.New(cp.baseURL,
		credhub.SkipTLSValidation(cp.skipTLSValidation),
		credhub.Auth(auth.Uaa(
			cp.clientID,
			cp.clientSecret,
			cp.username,
			cp.password,
			cp.accessToken,
			cp.refreshToken,
			false,
		)),
		credhub.AuthURL(cp.authURL),
	)

	return err
}

func (cp *CredhubProxy) auth() error {
	ch, err := credhub.New(cp.baseURL,
		credhub.SkipTLSValidation(cp.skipTLSValidation),
		credhub.Auth(auth.UaaPassword(cp.clientID, cp.clientSecret, cp.username, cp.password)))
	if err != nil {
		return err
	}
	authURL, err := ch.AuthURL()
	if err != nil {
		return err
	}

	uaaClient := uaa.Client{
		AuthURL: authURL,
		Client:  ch.Client(),
	}

	if cp.clientID != "" {
		cp.accessToken, err = uaaClient.ClientCredentialGrant(cp.clientID, cp.clientSecret)
		if err != nil {
			return err
		}
	} else {
		if cp.clientID == "" {
			// default value to be used
			cp.clientID = "credhub_cli"
		}
		cp.accessToken, cp.refreshToken, err = uaaClient.PasswordGrant(cp.clientID, cp.clientSecret, cp.username, cp.password)
		if err != nil {
			return err
		}
	}

	// ensure the .cv directory is created
	home, err := os.UserHomeDir()
	if err != nil {
		return err
	}

	homedir := filepath.Join(home, cp.configPath)
	if _, err := os.Stat(homedir); os.IsNotExist(err) {
		os.Mkdir(homedir, os.ModePerm)
	}

	if err != nil {
		return err
	}

	// write out the config file with the access token and refresh
	// write out as json for now
	// our config will just be a struct for now
	cvConfig := CVConfig{AccessToken: cp.accessToken, RefreshToken: cp.refreshToken, CredhubBaseURL: cp.baseURL, AuthURL: authURL, SkipTLSValidation: cp.skipTLSValidation}

	b, err := json.Marshal(&cvConfig)
	if err != nil {
		return err
	}

	err = ioutil.WriteFile(filepath.Join(homedir, "config.json"), b, os.ModePerm)
	if err != nil {
		return err
	}

	cp.client, err = credhub.New(cp.baseURL,
		credhub.SkipTLSValidation(cp.skipTLSValidation),
		credhub.Auth(auth.Uaa(
			cp.clientID,
			cp.clientSecret,
			cp.username,
			cp.password,
			cp.accessToken,
			cp.refreshToken,
			false,
		)),
	)
	if err != nil {
		return err
	}
	return nil
}
