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
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io/ioutil"
	"regexp"
	"sort"
	"strings"

	"code.cloudfoundry.org/credhub-cli/credhub"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/generate"
	"github.com/Venafi/vcert/pkg/certificate"
)

var ConfigFile = ".cv.conf"

type CV struct {
	configPath   string
	vcert        IVcertProxy
	credhub      ICredhubProxy
	configLoader ConfigLoader
}

func (c *CV) generateAndStoreCredhub(name string, v *GenerateAndStoreCommand, store bool) error {
	// parameters := models.GenerationParameters{
	parameters := generate.Certificate{
		// IncludeSpecial:   false,
		// ExcludeNumber:    f,
		// ExcludeUpper:     c.ExcludeUpper,
		// ExcludeLower:     c.ExcludeLower,
		KeyLength:        v.KeyLength,
		CommonName:       v.CommonName,
		Organization:     v.OrganizationName,
		OrganizationUnit: strings.Join(v.OrganizationalUnit, ","),
		Locality:         v.Locality,
		State:            v.State,
		Country:          v.Country,
		AlternativeNames: v.AlternativeName,
		ExtendedKeyUsage: v.ExtKeyUsage,
		KeyUsage:         v.KeyUsage,
		Duration:         v.Duration,
		Ca:               v.CA,
		SelfSign:         v.SelfSign,
		IsCA:             v.IsCA,
	}

	status("NOW GENERATING ON CREDHUB '%s'\n", name)
	certificate, err := c.credhub.generateCertificate(name, parameters, credhub.NoOverwrite)
	if err != nil {
		return err
	}

	if !store {
		return nil
	}

	status("NOW UPLOADING TO VENAFI '%s'\n", name)
	err = c.vcert.putCertificate(name, certificate.Value.Certificate, certificate.Value.PrivateKey)
	if err != nil {
		return err
	}

	return nil
}

func (c *CV) generateAndStore(name string, args *GenerateAndStoreCommand, store bool) error {
	status("NOW GENERATING ON VENAFI '%s'\n", name)
	// we assume that login has already been done on credhub
	cert, err := c.vcert.generate(args)
	if err != nil {
		return err
	}

	if !store {
		return nil
	}

	status("NOW UPLOADING TO CREDHUB '%s'\n", name)
	certName := name
	ca := ""
	certificate := cert.Certificate
	privateKey := cert.PrivateKey
	return c.credhub.putCertificate(certName, ca, certificate, privateKey)
}

func (cp *CV) deleteCert(name string) error {
	cert, err := cp.credhub.getCertificate(name)
	if err != nil {
		return err
	}
	certStr := cert.Value.Certificate
	tp, err := getThumbprint(certStr)
	if err != nil {
		return err
	}
	tp2 := hex.EncodeToString(tp[:])

	status("NOW DELETING FROM VENAFI '%s'\n", name)
	err = cp.vcert.revoke(tp2)
	if err != nil {
		return err
	}

	status("NOW DELETING FROM CREDHUB '%s'\n", name)
	return cp.credhub.deleteCert(name)
}

func (cp *CV) listBoth(args *ListCommand) ([]CertCompareData, error) {
	status("LISTING...\n")

	certInfo, err := cp.vcert.list(args.VenafiLimit, args.VenafiRoot)
	if err != nil {
		return []CertCompareData{}, err
	}

	items, err := cp.credhub.list()
	if err != nil {
		return []CertCompareData{}, err
	}

	certs := []credentials.CertificateMetadata{}
	for _, cert := range items {
		if strings.HasPrefix(cert.Name, args.CredhubRoot) {
			certs = append(certs, cert)
		}
	}

	var ct ComparisonStrategy
	switch {
	case args.ByThumbprint:
		ct = &ThumbprintStrategy{getCertificate: cp.credhub.getCertificate}
	case args.ByPath:
		ct = &PathStrategy{leftPrefix: joinRoot(args.VenafiRoot, args.VenafiPrefix, "\\"), rightPrefix: joinRoot(args.CredhubRoot, args.CredhubPrefix, "/")}
	default:
		ct = &CommonNameStrategy{}
	}
	data := compareCerts(ct, certInfo, certs, "", "")
	printCertsPretty(ct, data)
	e, ok := ct.(processErrors)
	if ok {
		for _, each := range e.getErrors() {
			errorf("%s\n", each)
		}
	}
	if len(certInfo) == args.VenafiLimit {
		errorf("The Venafi limit was hit, consider increasing -vlimit to increase the number of allowed records.\n")
	}

	return data, nil
}

func joinRoot(a, b, sep string) string {
	a = strings.TrimSuffix(a, sep)
	b = strings.TrimPrefix(b, sep)
	if a == "" {
		return b
	}
	return a + sep + b
}

func printCerts(data []CertCompareData) {
	for i, d := range data {
		verbose("%d %+v\n", i, d)
	}
}

func jsonMarshallToFile(data interface{}, filename string) error {
	bytes, err := json.Marshal(data)
	if err != nil {
		return err
	}

	return ioutil.WriteFile(filename, bytes, 0644)
}

type ComparisonStrategy interface {
	leftGet(l certificate.CertificateInfo) string
	rightGet(r credentials.CertificateMetadata) string
	leftTransform(in string) string
	rightTransform(in string) string
}

func buildCompareTransform(tct ComparisonStrategy) func(certificate.CertificateInfo, credentials.CertificateMetadata) int {
	return func(l certificate.CertificateInfo, r credentials.CertificateMetadata) int {
		return compareTransform(l, r, tct)
	}
}

func compareTransform(l certificate.CertificateInfo, r credentials.CertificateMetadata, tct ComparisonStrategy) int {
	commonName := tct.leftGet(l)
	credhubName := tct.rightGet(r)

	commonName = tct.leftTransform(commonName)
	credhubName = tct.rightTransform(credhubName)

	cmpVal := strings.Compare(commonName, credhubName)

	verbose("compare commonName %s with credhubName %s out %d\n", commonName, credhubName, cmpVal)
	return cmpVal
}

func compareCerts(ct ComparisonStrategy, certInfo []certificate.CertificateInfo, items []credentials.CertificateMetadata, leftPrefix, rightPrefix string) []CertCompareData {
	cc := &DefaultCertCollector{}

	cmpTransform := buildCompareTransform(ct)
	compareLists(certInfo, items, cmpTransform, cc, ct)

	ps, ok := ct.(postSort)
	if ok {
		ps.postSort(cc.data)
	}

	printCerts(cc.data)
	return cc.data
}

type CertCompareData struct {
	Left  *certificate.CertificateInfo
	Right *credentials.CertificateMetadata
}

func (c CertCompareData) String() string {
	out := ""
	if c.Left != nil {
		out += fmt.Sprintf(" Left:%+v ", *c.Left)
	} else {
		out += fmt.Sprintf(" Left: nil ")
	}
	if c.Right != nil {
		out += fmt.Sprintf(" Right:%+v ", *c.Right)
	} else {
		out += fmt.Sprintf(" Right: nil ")
	}
	return out
}

type DefaultCertCollector struct {
	data []CertCompareData
}

func (m *DefaultCertCollector) CertificateInfo(item certificate.CertificateInfo) {
	m.data = append(m.data, CertCompareData{Left: &item})
}
func (m *DefaultCertCollector) CertificateMetadata(item credentials.CertificateMetadata) {
	m.data = append(m.data, CertCompareData{Right: &item})
}
func (m *DefaultCertCollector) Equals(ci certificate.CertificateInfo, cm credentials.CertificateMetadata) {
	m.data = append(m.data, CertCompareData{Left: &ci, Right: &cm})
}

// CertCollector collects the comparison output
type CertCollector interface {
	// CertificateInfo handles a Venafi non-match
	CertificateInfo(certificate.CertificateInfo)
	// CertificateMetadata handles a Credhub non-match
	CertificateMetadata(credentials.CertificateMetadata)
	// Equals handles certificates that match
	Equals(certificate.CertificateInfo, credentials.CertificateMetadata)
}

func compareLists(
	l []certificate.CertificateInfo,
	r []credentials.CertificateMetadata,
	comparison func(certificate.CertificateInfo, credentials.CertificateMetadata) int,
	collector CertCollector,
	tct ComparisonStrategy) {
	sort.SliceStable(l, func(i, j int) bool {
		a := tct.leftGet(l[i])
		b := tct.leftGet(l[j])
		a = tct.leftTransform(a)
		b = tct.leftTransform(b)
		return a < b
	})

	sort.SliceStable(r, func(i, j int) bool {
		a := tct.rightGet(r[i])
		b := tct.rightGet(r[j])
		a = tct.rightTransform(a)
		b = tct.rightTransform(b)
		return a < b
	})

	// print the sorted lists using get
	for _, item := range l {
		after := tct.leftTransform(tct.leftGet(item))
		verbose("left %s", after)
	}
	for _, item := range r {
		after := tct.rightTransform(tct.rightGet(item))
		verbose("right %s", after)
	}

	compareSortedLists(l, r, comparison, collector)
}

func compareSortedLists(
	l []certificate.CertificateInfo,
	r []credentials.CertificateMetadata,
	comparison func(certificate.CertificateInfo, credentials.CertificateMetadata) int,
	collector CertCollector) {
	i := 0
	j := 0
	n1 := len(l)
	n2 := len(r)

	for i < n1 && j < n2 {
		cmp := comparison(l[i], r[j])
		if cmp < 0 {
			collector.CertificateInfo(l[i])
			i++
		} else if cmp == 0 {
			collector.Equals(l[i], r[j])
			i++
			j++
		} else {
			collector.CertificateMetadata(r[j])
			j++
		}
	}

	for i < n1 {
		collector.CertificateInfo(l[i])
		i++
	}

	for j < n2 {
		collector.CertificateMetadata(r[j])
		j++
	}
}

type CommonNameStrategy struct {
	leftPrefix  string
	rightPrefix string
}

func (t *CommonNameStrategy) leftGet(l certificate.CertificateInfo) string {
	return l.CN
}

func (t *CommonNameStrategy) rightGet(r credentials.CertificateMetadata) string {
	return r.Name
}

func (t *CommonNameStrategy) leftTransform(in string) string {
	return strings.TrimPrefix(in, t.leftPrefix)
}

func (t *CommonNameStrategy) rightTransform(in string) string {
	return credhubTransform(strings.TrimPrefix(in, t.rightPrefix))
}

func (t *CommonNameStrategy) headers() []string {
	return []string{"VENAFI", "CREDHUB"}
}

func (t *CommonNameStrategy) values(l *certificate.CertificateInfo, r *credentials.CertificateMetadata) []string {
	left := ""
	right := ""
	if l != nil {
		left = t.leftGet(*l)
	}
	if r != nil {
		right = t.rightGet(*r)
	}
	return []string{left, right}
}

type ThumbprintStrategy struct {
	leftPrefix      string
	rightPrefix     string
	getCertificate  func(name string) (credentials.Certificate, error)
	thumbprintCache map[string]string
	errors          []error
}

func (t *ThumbprintStrategy) leftGet(l certificate.CertificateInfo) string {
	return l.Thumbprint
}

func (t *ThumbprintStrategy) rightGet(r credentials.CertificateMetadata) string {
	in := r.Name

	// we check if this path is already in the thumbprint cache, and return it right away if it is
	i, ok := t.cache()[in]
	if ok {
		return i
	}
	// we do a get on cert name to get a cert
	cert, err := t.getCertificate(in)
	if err != nil {
		t.errors = append(t.errors, err)
	}

	// then, from the cert we calculate the thumbprint
	certStr := cert.Value.Certificate
	tp, err := getThumbprint(certStr)
	if err != nil {
		t.errors = append(t.errors, err)
	}
	tp2 := hex.EncodeToString(tp[:])
	verbose("thumbprint %s path %s", tp2, in)
	// then we store that thumbprint in the cache
	t.cache()[in] = tp2
	// and we return that thumbprint
	return tp2
}

func (t *ThumbprintStrategy) leftTransform(in string) string {
	return strings.ToUpper(strings.TrimPrefix(in, t.leftPrefix))
}

func (t *ThumbprintStrategy) rightTransform(in string) string {
	return strings.ToUpper(strings.TrimPrefix(in, t.leftPrefix))
}

func (t *ThumbprintStrategy) headers() []string {
	return []string{"VENAFI", "CREDHUB", "THUMBPRINT"}
}

func (t *ThumbprintStrategy) cache() map[string]string {
	if t.thumbprintCache == nil {
		t.thumbprintCache = map[string]string{}
	}
	return t.thumbprintCache
}

func (t *ThumbprintStrategy) getErrors() []error {
	return t.errors
}

func (t *ThumbprintStrategy) values(l *certificate.CertificateInfo, r *credentials.CertificateMetadata) []string {
	thumbprint := ""
	left := ""
	right := ""

	if l != nil {
		left = l.CN
		thumbprint = l.Thumbprint
	}
	if r != nil {
		right = r.Name

		i, ok := t.cache()[r.Name]
		if ok {
			thumbprint = i
		}
	}
	return []string{left, right, strings.ToLower(thumbprint)}
}

func (t *ThumbprintStrategy) postSort(l []CertCompareData) {
	cmp := func(i, j int) bool {
		a := l[i]
		b := l[j]
		if a.Left != nil && b.Left == nil {
			return false
		} else if a.Left == nil && b.Left != nil {
			return true
		} else if a.Left != nil && b.Left != nil {
			aID := a.Left.ID
			bID := b.Left.ID
			if aID < bID {
				return true
			}
		}

		if a.Right != nil && b.Right == nil {
			return false
		} else if a.Right == nil && b.Right != nil {
			return true
		}
		if a.Right != nil && b.Right != nil {
			aID := a.Right.Name
			bID := b.Right.Name
			return aID < bID
		}
		return !(a.Right == nil && b.Right == nil)
	}

	sort.SliceStable(l, func(i, j int) bool {
		return !cmp(i, j)
	})
}

type PathStrategy struct {
	leftPrefix  string
	rightPrefix string
}

func (t *PathStrategy) leftGet(l certificate.CertificateInfo) string {
	return l.ID
}

func (t *PathStrategy) rightGet(r credentials.CertificateMetadata) string {
	return r.Name
}

func (t *PathStrategy) leftTransform(in string) string {
	return t.normalize(in, t.leftPrefix)
}

func (t *PathStrategy) rightTransform(in string) string {
	return t.normalize(in, t.rightPrefix)
}

func (t *PathStrategy) normalize(in string, prefix string) string {
	prefix = strings.ReplaceAll(prefix, "\\", "/")
	in = strings.ReplaceAll(in, "\\", "/")
	return strings.TrimPrefix(strings.TrimPrefix(in, prefix), "/")
}

func (t *PathStrategy) leftDisplay(l certificate.CertificateInfo) string {
	return l.ID
}

func (t *PathStrategy) rightDisplay(r credentials.CertificateMetadata) string {
	return r.Name
}

func (t *PathStrategy) headers() []string {
	return []string{"VENAFI", "CREDHUB"}
}

func (t *PathStrategy) values(l *certificate.CertificateInfo, r *credentials.CertificateMetadata) []string {
	left := ""
	right := ""
	if l != nil {
		left = t.leftDisplay(*l)
	}
	if r != nil {
		right = t.rightDisplay(*r)
	}
	return []string{left, right}
}

type postSort interface {
	postSort(l []CertCompareData)
}

type processErrors interface {
	getErrors() []error
}

func prependPolicyRoot(zone string) string {
	zone = strings.TrimPrefix(zone, "\\")
	zone = strings.TrimPrefix(zone, "Policy\\")
	return prependVEDRoot("\\Policy\\" + zone)
}

func prependVEDRoot(zone string) string {
	zone = strings.TrimPrefix(zone, "\\")
	zone = strings.TrimPrefix(zone, "VED\\")
	return "\\VED\\" + zone
}

var TPPGeneratedNameRegex = regexp.MustCompile(`(.*)_[0-9]{2}[a-z]{3}[0-9]{2}_[A-Z]{2}[0-9]{2}`)

func removeTPPUploadSuffix(input string) string {
	return TPPGeneratedNameRegex.ReplaceAllString(input, "${1}")
}

func extractLastSegment(input string) string {
	split := strings.Split(input, "/")
	return split[len(split)-1]
}

func credhubTransform(input string) string {
	input = extractLastSegment(input)
	return removeTPPUploadSuffix(input)
}
