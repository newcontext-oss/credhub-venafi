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

package vcclient

import (
	"crypto/x509/pkix"
	"encoding/pem"
	"fmt"
	"strings"
	"time"

	"github.com/Venafi/vcert"
	"github.com/Venafi/vcert/pkg/certificate"
	"github.com/Venafi/vcert/pkg/endpoint"
	"github.com/newcontext-oss/credhub-venafi/output"
)

// IVcertProxy defines the interface for proxies that manage requests to vcert
type IVcertProxy interface {
	PutCertificate(certName string, cert string, privateKey string) error
	List(vlimit int, zone string) ([]certificate.CertificateInfo, error)
	RetrieveCertificateByThumbprint(thumprint string) (*certificate.PEMCollection, error)
	Login() error
	Revoke(thumbprint string) error
	Generate(args *GenerateAndStoreCommand) (*certificate.PEMCollection, error)
}

// VcertProxy contains the necessary config information for a vcert proxy
type VcertProxy struct {
	Username      string
	Password      string
	Zone          string
	Client        endpoint.Connector
	BaseURL       string
	ConnectorType string
}

// PutCertificate uploads a certificate to vcert
func (v *VcertProxy) PutCertificate(certName string, cert string, privateKey string) error {
	importReq := &certificate.ImportRequest{
		// if PolicyDN is empty, it is taken from cfg.Zone
		ObjectName:      certName,
		CertificateData: cert,
		PrivateKeyData:  privateKey,
		// Password:        "newPassw0rd!",
		Reconcile: false,
	}

	importResp, err := v.Client.ImportCertificate(importReq)
	if err != nil {
		return err
	}
	output.Verbose("%+v", importResp)
	return nil
}

// List retrieves the list of certificates from vcert
func (v *VcertProxy) List(limit int, zone string) ([]certificate.CertificateInfo, error) {
	output.Info("vcert list from proxy")

	v.Client.SetZone(prependVEDRoot(zone))
	filter := endpoint.Filter{Limit: &limit, WithExpired: true}
	certInfo, err := v.Client.ListCertificates(filter)
	if err != nil {
		return []certificate.CertificateInfo{}, err
	}
	output.Verbose("certInfo %+v", certInfo)
	for a, b := range certInfo {
		output.Verbose("cert %+v %+v\n", a, b)
	}
	return certInfo, nil
}

// RetrieveCertificateByThumbprint fetches a certificate from vcert by the thumbprint
func (v *VcertProxy) RetrieveCertificateByThumbprint(thumprint string) (*certificate.PEMCollection, error) {
	pickupReq := &certificate.Request{
		Thumbprint: thumprint,
		Timeout:    180 * time.Second,
	}

	return v.Client.RetrieveCertificate(pickupReq)
}

// Login creates a session with the TPP server
func (v *VcertProxy) Login() error {
	auth := endpoint.Authentication{
		User:     v.Username,
		Password: v.Password,
	}
	var connectorType endpoint.ConnectorType

	switch v.ConnectorType {
	case "tpp":
		connectorType = endpoint.ConnectorTypeTPP
	default:
		return fmt.Errorf("connector type '%s' not found", v.ConnectorType)
	}
	conf := vcert.Config{
		Credentials:   &auth,
		BaseUrl:       v.BaseURL,
		Zone:          v.Zone,
		ConnectorType: connectorType,
	}
	c, err := vcert.NewClient(&conf)
	if err != nil {
		return fmt.Errorf("could not connect to endpoint: %s", err)
	}
	v.Client = c

	return nil
}

// Revoke revokes a certificate in vcert (delete is not available via the api)
func (v *VcertProxy) Revoke(thumbprint string) error {
	revokeReq := &certificate.RevocationRequest{
		// CertificateDN: requestID,
		Thumbprint: thumbprint,
		Reason:     "key-compromise",
		Comments:   "revocation comment below",
		Disable:    false,
	}

	err := v.Client.RevokeCertificate(revokeReq)
	if err != nil {
		return err
	}

	output.Verbose("Successfully submitted revocation request for thumbprint %s", thumbprint)
	return nil
}

// Generate generates a certificate in vcert
func (v *VcertProxy) Generate(args *GenerateAndStoreCommand) (*certificate.PEMCollection, error) {
	req, err := buildGenerateRequest(args)
	if err != nil {
		return nil, err
	}

	requestID, privateKey, err := sendCertificateRequest(v.Client, req)
	if err != nil {
		return nil, err
	}

	pickupReq := &certificate.Request{
		PickupID: requestID,
		Timeout:  180 * time.Second,
	}

	pcc, err := v.Client.RetrieveCertificate(pickupReq)
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

	output.Verbose("Successfully submitted certificate request. Will pickup certificate by ID %s", requestID)
	return requestID, privateKey, nil
}

// PrependPolicyRoot adds \Policy\ to the front of the zone string
func PrependPolicyRoot(zone string) string {
	zone = strings.TrimPrefix(zone, "\\")
	zone = strings.TrimPrefix(zone, "Policy\\")
	return prependVEDRoot("\\Policy\\" + zone)
}

func prependVEDRoot(zone string) string {
	zone = strings.TrimPrefix(zone, "\\")
	zone = strings.TrimPrefix(zone, "VED\\")
	return "\\VED\\" + zone
}
