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
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"time"

	"github.com/Venafi/vcert"
	"github.com/Venafi/vcert/pkg/certificate"
	"github.com/Venafi/vcert/pkg/endpoint"
)

// IVcertProxy defines the interface for proxies that manage requests to vcert
type IVcertProxy interface {
	putCertificate(certName string, cert string, privateKey string) error
	list(vlimit int, zone string) ([]certificate.CertificateInfo, error)
	retrieveCertificateByThumbprint(thumprint string) (*certificate.PEMCollection, error)
	login() error
	revoke(thumbprint string) error
	generate(args *GenerateAndStoreCommand) (*certificate.PEMCollection, error)
}

// VcertProxy contains the necessary config information for a vcert proxy
type VcertProxy struct {
	username      string
	password      string
	zone          string
	client        endpoint.Connector
	baseURL       string
	connectorType string
}

func (v *VcertProxy) putCertificate(certName string, cert string, privateKey string) error {
	importReq := &certificate.ImportRequest{
		// if PolicyDN is empty, it is taken from cfg.Zone
		ObjectName:      certName,
		CertificateData: cert,
		PrivateKeyData:  privateKey,
		// Password:        "newPassw0rd!",
		Reconcile: false,
	}

	importResp, err := v.client.ImportCertificate(importReq)
	if err != nil {
		return err
	}
	verbose("%+v", importResp)
	return nil
}

func (v *VcertProxy) list(limit int, zone string) ([]certificate.CertificateInfo, error) {
	info("vcert list from proxy")

	v.client.SetZone(prependVEDRoot(zone))
	filter := endpoint.Filter{Limit: &limit, WithExpired: true}
	certInfo, err := v.client.ListCertificates(filter)
	if err != nil {
		return []certificate.CertificateInfo{}, err
	}
	verbose("certInfo %+v", certInfo)
	for a, b := range certInfo {
		verbose("cert %+v %+v\n", a, b)
	}
	return certInfo, nil
}

func (v *VcertProxy) retrieveCertificateByThumbprint(thumprint string) (*certificate.PEMCollection, error) {
	pickupReq := &certificate.Request{
		Thumbprint: thumprint,
		Timeout:    180 * time.Second,
	}

	return v.client.RetrieveCertificate(pickupReq)
}

func (v *VcertProxy) login() error {
	auth := endpoint.Authentication{
		User:     v.username,
		Password: v.password,
	}
	var connectorType endpoint.ConnectorType

	switch v.connectorType {
	case "tpp":
		connectorType = endpoint.ConnectorTypeTPP
	default:
		return fmt.Errorf("Connector type '%s' not found", v.connectorType)
	}
	conf := vcert.Config{
		Credentials:   &auth,
		BaseUrl:       v.baseURL,
		Zone:          v.zone,
		ConnectorType: connectorType,
	}
	c, err := vcert.NewClient(&conf)
	if err != nil {
		return fmt.Errorf("could not connect to endpoint: %s", err)
	}
	v.client = c

	return nil
}

func (v *VcertProxy) revoke(thumbprint string) error {
	revokeReq := &certificate.RevocationRequest{
		// CertificateDN: requestID,
		Thumbprint: thumbprint,
		Reason:     "key-compromise",
		Comments:   "revocation comment below",
		Disable:    false,
	}

	err := v.client.RevokeCertificate(revokeReq)
	if err != nil {
		return err
	}

	verbose("Successfully submitted revocation request for thumbprint %s", thumbprint)
	return nil
}

func (v *VcertProxy) generate(args *GenerateAndStoreCommand) (*certificate.PEMCollection, error) {
	req, err := buildGenerateRequest(args)
	if err != nil {
		return nil, err
	}

	requestID, privateKey, err := sendCertificateRequest(v.client, req)

	pickupReq := &certificate.Request{
		PickupID: requestID,
		Timeout:  180 * time.Second,
	}

	pcc, err := v.client.RetrieveCertificate(pickupReq)
	if err != nil {
		return nil, fmt.Errorf("could not retrieve certificate using requestId %s: %s", requestID, err)
	}
	pcc.PrivateKey = privateKey
	return pcc, nil
}

func buildGenerateRequest(v *GenerateAndStoreCommand) (*certificate.Request, error) {
	r := &certificate.Request{}
	r.FriendlyName = v.Name

	subject := pkix.Name{}
	if v.CommonName != "" {
		subject.CommonName = v.CommonName
	}
	if v.OrganizationName != "" {
		subject.Organization = []string{v.OrganizationName}
	}
	if len(v.SANDNS) != 0 {
		r.DNSNames = v.SANDNS
	}
	r.KeyCurve = v.KeyCurve
	if len(v.OrganizationalUnit) > 0 {
		subject.OrganizationalUnit = v.OrganizationalUnit
	}
	if v.Country != "" {
		subject.Country = []string{v.Country}
	}
	if v.State != "" {
		subject.Province = []string{v.State}
	}
	if v.Locality != "" {
		subject.Locality = []string{v.Locality}
	}
	if len(v.SANEmail) > 0 {
		r.EmailAddresses = v.SANEmail
	}
	if len(v.SANIP) > 0 {
		r.IPAddresses = v.SANIP
	}
	if v.KeyPassword == "" {
		r.KeyPassword = v.KeyPassword
	}
	r.Subject = subject
	return r, nil
}

func sendCertificateRequest(c endpoint.Connector, enrollReq *certificate.Request) (requestID string, privateKey string, err error) {
	err = c.GenerateRequest(nil, enrollReq)
	if err != nil {
		return "", "", err
	}

	requestID, err = c.RequestCertificate(enrollReq)
	if err != nil {
		return "", "", err
	}

	pemBlock, err := certificate.GetPrivateKeyPEMBock(enrollReq.PrivateKey)
	if err != nil {
		return "", "", err
	}
	privateKey = string(pem.EncodeToMemory(pemBlock))

	verbose("Successfully submitted certificate request. Will pickup certificate by ID %s", requestID)
	return requestID, privateKey, nil
}
