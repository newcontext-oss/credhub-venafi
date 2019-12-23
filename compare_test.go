package main

import (
	"encoding/json"
	"fmt"
	"io/ioutil"
	"path/filepath"
	"strings"
	"testing"

	"code.cloudfoundry.org/credhub-cli/credhub/credentials"
	"code.cloudfoundry.org/credhub-cli/credhub/credentials/values"
	"github.com/Venafi/vcert/pkg/certificate"
)

func TestCmpMethod(t *testing.T) {
	comparison := func(l string, r string) int {
		return strings.Compare(l, r)
	}
	a := "a"
	b := "b"
	fmt.Printf("%s and %s cmp  %b", a, b, comparison(a, b))
	// -1
}

func TestCmp3(t *testing.T) {
	comparison := func(l string, r string) int {
		return strings.Compare(l, r)
	}
	comparesides2([]string{"a", "b"}, []string{"c", "d"}, comparison)
	comparesides2([]string{"a", "c"}, []string{"b", "d"}, comparison)
	comparesides2([]string{}, []string{"a", "b", "c", "d"}, comparison)
	comparesides2([]string{"a", "b", "c", "d"}, []string{}, comparison)
	comparesides2([]string{"a"}, []string{"b", "c", "d"}, comparison)
	comparesides2([]string{"a", "b", "c"}, []string{"d"}, comparison)
}

type TestCertCollector struct {
	values   []string
	leftGet  func(certificate.CertificateInfo) string
	rightGet func(credentials.CertificateMetadata) string
}

func (m *TestCertCollector) CertificateInfo(ci certificate.CertificateInfo) {
	if m.leftGet == nil {
		m.leftGet = func(l certificate.CertificateInfo) string {
			return l.CN
		}
	}
	fmt.Printf(".")
	m.values = append(m.values, m.leftGet(ci))
}

func (m *TestCertCollector) CertificateMetadata(cm credentials.CertificateMetadata) {
	if m.rightGet == nil {
		m.rightGet = func(r credentials.CertificateMetadata) string {
			return r.Name
		}
	}
	fmt.Printf(".")
	m.values = append(m.values, m.rightGet(cm))
}

func (m *TestCertCollector) Equals(ci certificate.CertificateInfo, cm credentials.CertificateMetadata) {
	fmt.Printf(".")
	m.values = append(m.values, m.leftGet(ci)+"="+m.rightGet(cm))
}

func TestExtractLastSegment(t *testing.T) {
	lastsegment := extractLastSegment("/thelast")
	if lastsegment != "thelast" {
		t.Errorf("last segment value was %s instead of thelast", lastsegment)
	}
}

type TestStructMember struct {
	a string
}

type TestStruct struct {
	left  *TestStructMember
	right *TestStructMember
}

func TestJsonSerialize(t *testing.T) {
	// a := []TestStruct{TestStruct{left:}}
	a := []TestStruct{TestStruct{}}
	bytes, err := json.Marshal(a)
	if err != nil {
		fmt.Println("e", err)
	}
	fmt.Println("s", string(bytes))
}

func aTestJsonDeserialize(t *testing.T) {
	data := []CertCompareData{}

	dat, err := ioutil.ReadFile("/tmp/dat1")
	if err != nil {
		fmt.Println("err", err)
		return
	}

	err = json.Unmarshal(dat, &data)
	if err != nil {
		fmt.Println("e", err)
		return
	}
	fmt.Printf("zoo %+v\n", data)

	bytes, err := json.Marshal(data)
	if err != nil {
		fmt.Println("e", err)
		return
	}

	err = ioutil.WriteFile("/tmp/dat2", bytes, 0644)

	if err != nil {
		fmt.Println("e", err)
		return
	}
}

func jsonUnmarshallFromFile(v interface{}, filename string) {
	path := filepath.Join("testdata", filename)
	dat, err := ioutil.ReadFile(path)
	if err != nil {
		fmt.Println("err", err)
		return
	}

	err = json.Unmarshal(dat, v)
	if err != nil {
		fmt.Println("e", err)
		return
	}
}

func TestCompareCerts(t *testing.T) {
	certInfo := []certificate.CertificateInfo{}
	items := []credentials.CertificateMetadata{}
	jsonUnmarshallFromFile(&certInfo, "certinfo.json")
	jsonUnmarshallFromFile(&items, "chitems.json")

	ct := CommonNameStrategy{}
	certCompare := compareCerts(&ct, certInfo, items, "", "")
	assertLenEquals(t, len(certCompare), 4)
	assertStringEquals(t, certCompare[0].Left.CN, "TestCertb")
	assertStringContains(t, certCompare[0].Right.Name, "TestCertb")
	assertStringEquals(t, certCompare[1].Left.CN, "TestCommonName")
	assertStringContains(t, certCompare[1].Right.Name, "TestCommonName")
	assertTrue(t, certCompare[2].Left == nil)
	assertStringEquals(t, certCompare[2].Right.Name, "/aname")
	assertStringEquals(t, certCompare[3].Left.CN, "localhost")
	assertTrue(t, certCompare[3].Right == nil)
}

func assertTrue(t *testing.T, result bool) {
	t.Helper()
	if !result {
		t.Errorf("expected true but was false")
	}
}

func assertStringEquals(t *testing.T, expected, actual string) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected '%s' but was '%s'", expected, actual)
	}
}

func assertStringContains(t *testing.T, actual, shouldContain string) {
	t.Helper()
	if !strings.Contains(actual, shouldContain) {
		t.Errorf("expected to contain '%s' but was '%s'", shouldContain, actual)
	}
}

func assertLenEquals(t *testing.T, expected, actual int) {
	t.Helper()
	if expected != actual {
		t.Errorf("expected length of %d but was %d", expected, actual)
	}
}

func TestComparePathPrefixTransform(t *testing.T) {
	tctWithPrefix := PathStrategy{leftPrefix: "\\FRED", rightPrefix: "\\VED"}
	tctEmpty := PathStrategy{}

	tests := []struct {
		tct   ComparisonStrategy
		left  []string
		right []string
		out   []string
	}{
		{
			&tctEmpty,
			[]string{
				"\\VED\\Policy\\Certificates\\Division 3\\TestCerta",
				"\\VED\\Policy\\Certificates\\Division 3\\localhost",
				"\\VED\\Policy\\Certificates\\Division 3\\TestCertb"},
			[]string{
				"\\VED\\Policy\\Certificates\\Division 3\\localhost",
				"\\VED\\Policy\\Certificates\\Division 3\\TestCerta",
				"\\VED\\Policy\\Certificates\\Division 3\\TestCertb"},
			[]string{
				"\\VED\\Policy\\Certificates\\Division 3\\TestCerta=\\VED\\Policy\\Certificates\\Division 3\\TestCerta",
				"\\VED\\Policy\\Certificates\\Division 3\\TestCertb=\\VED\\Policy\\Certificates\\Division 3\\TestCertb",
				"\\VED\\Policy\\Certificates\\Division 3\\localhost=\\VED\\Policy\\Certificates\\Division 3\\localhost"}},
		{
			&tctWithPrefix,
			[]string{
				"\\FRED\\Policy\\Certificates\\Division 3\\TestCerta",
				"\\FRED\\Policy\\Certificates\\Division 3\\localhost",
				"\\FRED\\Policy\\Certificates\\Division 3\\TestCertb"},
			[]string{
				"\\VED\\Policy\\Certificates\\Division 3\\localhost",
				"\\VED\\Policy\\Certificates\\Division 3\\TestCerta",
				"\\VED\\Policy\\Certificates\\Division 3\\TestCertb"},
			[]string{
				"\\FRED\\Policy\\Certificates\\Division 3\\TestCerta=\\VED\\Policy\\Certificates\\Division 3\\TestCerta",
				"\\FRED\\Policy\\Certificates\\Division 3\\TestCertb=\\VED\\Policy\\Certificates\\Division 3\\TestCertb",
				"\\FRED\\Policy\\Certificates\\Division 3\\localhost=\\VED\\Policy\\Certificates\\Division 3\\localhost"}},
	}

	runTests := func() {
		for _, test := range tests {
			left := []certificate.CertificateInfo{}
			for _, item := range test.left {
				left = append(left, certificate.CertificateInfo{ID: item})
			}
			right := []credentials.CertificateMetadata{}
			for _, item := range test.right {
				right = append(right, credentials.CertificateMetadata{Name: item})
			}

			comparison := buildCompareTransform(test.tct)

			compare := func(
				l []certificate.CertificateInfo,
				r []credentials.CertificateMetadata,
				comparison func(certificate.CertificateInfo, credentials.CertificateMetadata) int, tc CertCollector) {
				compareLists(l, r, comparison, tc, test.tct)
			}

			tc := &TestCertCollector{leftGet: test.tct.leftGet, rightGet: test.tct.rightGet}
			compare(left, right, comparison, tc)

			assertStringSliceEqual(t, test.out, tc.values)
		}
	}

	runTests()
}

func TestCompareTransformCommonName(t *testing.T) {
	tests := []struct {
		left  []string
		right []string
		out   []string
	}{
		{[]string{"TestCommonName"}, []string{"/booyah/TestCommonName_20nov25_DE13"}, []string{"TestCommonName=/booyah/TestCommonName_20nov25_DE13"}},
		{[]string{"TestCommonName", "abc", "zoo"}, []string{"/booyah/TestCommonName_20nov25_DE13", "def", "/zoo"}, []string{"TestCommonName=/booyah/TestCommonName_20nov25_DE13", "abc", "def", "zoo=/zoo"}},
	}

	tct := CommonNameStrategy{}
	comparison := buildCompareTransform(&tct)

	compare := func(
		l []certificate.CertificateInfo,
		r []credentials.CertificateMetadata,
		comparison func(certificate.CertificateInfo, credentials.CertificateMetadata) int, tc CertCollector) {
		compareLists(l, r, comparison, tc, &tct)
	}
	runTests := func() {
		for _, test := range tests {
			left := []certificate.CertificateInfo{}
			for _, item := range test.left {
				left = append(left, certificate.CertificateInfo{CN: item})
			}
			right := []credentials.CertificateMetadata{}
			for _, item := range test.right {
				right = append(right, credentials.CertificateMetadata{Name: item})
			}
			tc := &TestCertCollector{leftGet: tct.leftGet, rightGet: tct.rightGet}
			compare(left, right, comparison, tc)
			assertStringSliceEqual(t, test.out, tc.values)
		}
	}

	// run sorting the list beforehand
	runTests()

	compare = func(
		l []certificate.CertificateInfo,
		r []credentials.CertificateMetadata,
		comparison func(certificate.CertificateInfo, credentials.CertificateMetadata) int, tc CertCollector) {
		compareSortedLists(l, r, comparison, tc)
	}

	// run sorting the list beforehand output should be the same, but won't be if sorting causes issues
	runTests()
}

func TestCompareFunc(t *testing.T) {
	tests := []struct {
		left  []string
		right []string
		out   []string
	}{
		{[]string{"a", "b"}, []string{"c", "d"}, []string{"a", "b", "c", "d"}},
		{[]string{"a", "c"}, []string{"b", "d"}, []string{"a", "b", "c", "d"}},
		{[]string{}, []string{"a", "b", "c", "d"}, []string{"a", "b", "c", "d"}},
		{[]string{"a", "b", "c", "d"}, []string{}, []string{"a", "b", "c", "d"}},
		{[]string{"a"}, []string{"b", "c", "d"}, []string{"a", "b", "c", "d"}},
		{[]string{"a", "b", "c"}, []string{"d"}, []string{"a", "b", "c", "d"}},
	}

	comparison := func(l certificate.CertificateInfo, r credentials.CertificateMetadata) int {
		return strings.Compare(l.CN, r.Name)
	}

	compare := func(
		l []certificate.CertificateInfo,
		r []credentials.CertificateMetadata,
		comparison func(certificate.CertificateInfo, credentials.CertificateMetadata) int, tc CertCollector) {
		compareSortedLists(l, r, comparison, tc)
	}
	runTests := func() {
		for _, test := range tests {
			left := []certificate.CertificateInfo{}
			for _, item := range test.left {
				left = append(left, certificate.CertificateInfo{CN: item})
			}
			right := []credentials.CertificateMetadata{}
			for _, item := range test.right {
				right = append(right, credentials.CertificateMetadata{Name: item})
			}
			tc := &TestCertCollector{}
			compare(left, right, comparison, tc)
			assertStringSliceEqual(t, test.out, tc.values)
		}
	}

	runTests()
}

func assertStringSliceEqual(t *testing.T, a, b []string) {
	if len(a) != len(b) {
		t.Errorf("slices not of equals size expected size %d actual size %d", len(a), len(b))
		t.Errorf("slices not equal expected %+v actual %+v", a, b)
		return
	}
	for i, v := range a {
		if v != b[i] {
			t.Errorf("slices not equal expected %+v actual %+v", a, b)
			return
		}
	}
}

func comparesides2(l, r []string, comparison func(string, string) int) {
	// Initial indexes of first and second subarrays
	i := 0
	j := 0
	n1 := len(l)
	n2 := len(r)

	for i < n1 && j < n2 {
		if comparison(l[i], r[j]) < 0 {
			fmt.Printf("l[i] %s\n", l[i])
			i++
		} else {
			fmt.Printf("r[j] %s\n", r[j])
			j++
		}
	}

	for i < n1 {
		// arr[k] = L[i];
		fmt.Printf("*l[i] %s\n", l[i])
		i++
	}

	for j < n2 {
		// arr[k] = R[j];
		fmt.Printf("*r[j] %s\n", r[j])
		j++
	}
}

func TestTrimPrefix(t *testing.T) {
	result := strings.TrimPrefix("sammadiṭṭhi", "samma")
	if result != "diṭṭhi" {
		t.Error("did not match")
	}
}

func TestRegexReplace(t *testing.T) {
	if removeTPPUploadSuffix("TestCommonName_20nov25_DE13") != "TestCommonName" {
		t.Error("did not match")
	}
	if removeTPPUploadSuffix("TestCommonName_20nov25_DE1") != "TestCommonName_20nov25_DE1" {
		t.Error("did not match")
	}
}

func TestCompareAndTransformThumbprint(t *testing.T) {
	c := ThumbprintStrategy{}
	// assertStringEquals(t, credname, "credname")
	c.getCertificate = func(name string) (credentials.Certificate, error) {
		return credentials.Certificate{Value: values.Certificate{Certificate: GetCert()}}, nil
	}
	credname := c.rightGet(credentials.CertificateMetadata{Name: "credname"})
	assertStringEquals(t, "ebdbe32ef98991695958ea2510287f0e6c52a483", credname)
	// output := c.rightTransform("credname")
	// fmt.Println("s", output)
}

func TestJoinRoot(t *testing.T) {
	assertStringEquals(t, "a/b", joinRoot("a", "b", "/"))
	assertStringEquals(t, "a/b", joinRoot("a/", "/b", "/"))
	assertStringEquals(t, "a/b", joinRoot("a/", "b", "/"))
	assertStringEquals(t, "a/b", joinRoot("a", "/b", "/"))

	assertStringEquals(t, "a\\b", joinRoot("a", "b", "\\"))
	assertStringEquals(t, "a\\b", joinRoot("a\\", "\\b", "\\"))
	assertStringEquals(t, "a\\b", joinRoot("a\\", "b", "\\"))
	assertStringEquals(t, "a\\b", joinRoot("a", "\\b", "\\"))
}

type CredhubProxyMock struct {
	CredhubProxy
	returnlist []credentials.CertificateMetadata
}

func (cp *CredhubProxyMock) list() ([]credentials.CertificateMetadata, error) {
	return cp.returnlist, nil
}

type VcertProxyMock struct {
	VcertProxy
	retCerts []certificate.CertificateInfo
}

func (v *VcertProxyMock) list(vlimit int, zone string) ([]certificate.CertificateInfo, error) {
	return v.retCerts, nil
}

func TestCVListBoth(t *testing.T) {
	tests := []struct {
		left        []string
		right       []string
		leftPrefix  string
		rightPrefix string
		out         []string
	}{
		{[]string{"a", "b"}, []string{"a", "b"}, "", "", []string{"a=a", "b=b"}},
		{[]string{"/a", "b"}, []string{"a", "b"}, "", "", []string{"/a=a", "b=b"}},
		{[]string{"a", "b"}, []string{"a", "/b"}, "", "", []string{"a=a", "b=/b"}},
		{[]string{"a", "b"}, []string{"/b", "a"}, "", "", []string{"a=a", "b=/b"}},
		{[]string{"/z/a", "b"}, []string{"/b", "a"}, "/z/", "", []string{"/z/a=a", "b=/b"}},
	}

	for _, test := range tests {
		left := []certificate.CertificateInfo{}
		for _, item := range test.left {
			left = append(left, certificate.CertificateInfo{ID: item})
		}
		right := []credentials.CertificateMetadata{}
		for _, item := range test.right {
			right = append(right, credentials.CertificateMetadata{Name: item})
		}

		ch := CredhubProxyMock{returnlist: right}
		v := VcertProxyMock{retCerts: left}
		c := CV{credhub: &ch, vcert: &v}
		l := &ListCommand{VenafiPrefix: test.leftPrefix, CredhubPrefix: test.rightPrefix, ByPath: true}
		r, err := c.listBoth(l)
		assertTrue(t, err == nil)
		s := []string{}
		for _, i := range r {
			l := ""
			r := ""
			if i.Left != nil {
				l = i.Left.ID
			}
			if i.Right != nil {
				r = i.Right.Name
			}
			// fmt.Println("s", i)
			s = append(s, fmt.Sprintf("%s=%s", l, r))
		}
		fmt.Println("s", s)

		// tc := &TestCertCollector{}
		// compare(left, right, comparison, tc)
		assertStringSliceEqual(t, test.out, s)
	}
}

func GetCA() string {
	return "-----BEGIN CERTIFICATE-----\nMIIDTTCCAjWgAwIBAgIULxxoB3zfye0MzzRQGtKtw8CC2p4wDQYJKoZIhvcNAQEL\nBQAwGjEYMBYGA1UEAwwPZm9vX2NlcnRpZmljYXRlMB4XDTE3MTEyMTE2MjQ1NFoX\nDTE4MTEyMTE2MjQ1NFowGjEYMBYGA1UEAwwPZm9vX2NlcnRpZmljYXRlMIIBIjAN\nBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAvgmVFC08CaGlp2ENKc5mym9BkcEL\nE2030VXTJJSiLH1Py3s79rbJL4F/loxAMSbQCuKIADZZ4Wu7Xp8bFY92u0nNrAuG\nXw1omhfF5UTi1vewRG8inxdJZs9vxsiXZWI6WzUpvJZeCMvvKlkhA/C+GwjpIdRg\n3fVZ44JvdZOx4cDUagRkDRcsHABp/ip19xhfZGLwFuJw1wd5kxZmZKCreoel+b/5\nBaQyFD0L9OkSfq1Y/nMqOdIbXEnpeg8sawhQan0s7G98MTsvDF14jnucCECLO1fg\n1YpxoBTDkNlrIPq4G8UO4+GNz5FJEIBGOsiRmEn0VjFEpZ3k+t/Nkf/b6wIDAQAB\no4GKMIGHMB0GA1UdDgQWBBQ3ZlJJaG9Brzf3IM6tWsMJce6YIDBVBgNVHSMETjBM\ngBQ3ZlJJaG9Brzf3IM6tWsMJce6YIKEepBwwGjEYMBYGA1UEAwwPZm9vX2NlcnRp\nZmljYXRlghQvHGgHfN/J7QzPNFAa0q3DwILanjAPBgNVHRMBAf8EBTADAQH/MA0G\nCSqGSIb3DQEBCwUAA4IBAQAlbxUF4Eaz0tXSo7oM02Mt3YqhuP7XZpZE5KYpn5qE\nutYzJdSJeMsfUpZcmv1pbZ4uepxgBxQKssKRmglzEMX2wxl9WyEPxKkyLTX+XCX9\nVd6IBi5Pft6v2u94bKlGZKigNojGfbzXDYuSU6SAud5GD77RM1vx/pPAa2eG8qSX\nOcGQAtHrcSAvl58IqXAmci3akNKN4G5PxNoze5lQ25umQbHTlwvOMwFgPSXseYvm\n/f98b+Q6lIdklw6g3XWUCmTkscRM+5mvb+1FKHWU8KiXN7CM+ONXjudO8Ixyyion\npBumFgiA2FQXUpunDCv38dccPb8y/EyhRSQyx+olXqo+\n-----END CERTIFICATE-----"
}

func GetCert() string {
	return "-----BEGIN CERTIFICATE-----\nMIIDSjCCAjKgAwIBAgIUdpQ3G/AnIilrPAsvMz3Zf9VnvWgwDQYJKoZIhvcNAQEL\nBQAwGjEYMBYGA1UEAwwPZm9vX2NlcnRpZmljYXRlMB4XDTE3MTEyMTE2MjUyMFoX\nDTE4MTEyMTE2MjUyMFowGjEYMBYGA1UEAwwPZm9vX2NlcnRpZmljYXRlMIIBIjAN\nBgkqhkiG9w0BAQEFAAOCAQ8AMIIBCgKCAQEAwqIrV8HpCuPyuJ6VvyG7gVhYJGAO\nX4zhclxkTAKT5rkE4Lfj048GZsDghK+pHs+tVotfyrJzYGJoEBTn9Wy7kP5pQmLR\nF54imDztep15OlyoJmLZfRgct/8Kyxkjgg3PKVw68IiNhnTlYaw4CAyZ/13mvw2c\nWIYlag9LV5R2ifcyubaYllxJhdWSXrcbYxrts1kRsUQTo99jJzKu71meLigMryaM\nry8xvjv1X8Yjq3s3Lud6gWZ6BuaaaVVIjI9clGgR1MkgKJgVkWjNzDRiCxYnq1LH\nCho9bgKgiY4p604zPk9Mw4FhtCbOim6HOsHTimONZXfDNmfsJ9wJefA0UwIDAQAB\no4GHMIGEMB0GA1UdDgQWBBTyAOrrFMy88bGgEBVI4PRGD4b02jBVBgNVHSMETjBM\ngBQ3ZlJJaG9Brzf3IM6tWsMJce6YIKEepBwwGjEYMBYGA1UEAwwPZm9vX2NlcnRp\nZmljYXRlghQvHGgHfN/J7QzPNFAa0q3DwILanjAMBgNVHRMBAf8EAjAAMA0GCSqG\nSIb3DQEBCwUAA4IBAQBC1x2+E35y+iX3Mu+SWD1I3RNTGE3qKdUqj+O+QeavqCRQ\n01nolxFaSvrM/4znAlWukfp9lCOHl8foD3vHQ+meW+PlLIH9HlBjn9T3c6h4p8EQ\niYV93tyCmUlPdtzW7k4Onl3IroNNHem9Uj+OSZxGtw35YU84T+hM1kaDKtZeS1je\nFWF1W8DCORxD2rFXFwe2nJd6SSeF3KWzuKAKDqJ7CmbdRb1TtgjUym6X55SQfW2a\ndwNE+9ztMBQm4ERhwMU/NMx14UjsOPvNjF1VVei52qQ2ce7c1vgW1RI2cYFgV8q8\noFjMdJePy7eLbGRaW7Jpdy9MOiEZOj513lT5MBGk\n-----END CERTIFICATE-----"
}

func GetPrivateKey() string {
	return "-----BEGIN RSA PRIVATE KEY----- fake\nMIIEpQIBAAKCAQEAwqIrV8HpCuPyuJ6VvyG7gVhYJGAOX4zhclxkTAKT5rkE4Lfj\n048GZsDghK+pHs+tVotfyrJzYGJoEBTn9Wy7kP5pQmLRF54imDztep15OlyoJmLZ\nfRgct/8Kyxkjgg3PKVw68IiNhnTlYaw4CAyZ/13mvw2cWIYlag9LV5R2ifcyubaY\nllxJhdWSXrcbYxrts1kRsUQTo99jJzKu71meLigMryaMry8xvjv1X8Yjq3s3Lud6\ngWZ6BuaaaVVIjI9clGgR1MkgKJgVkWjNzDRiCxYnq1LHCho9bgKgiY4p604zPk9M\nw4FhtCbOim6HOsHTimONZXfDNmfsJ9wJefA0UwIDAQABAoIBAEwsTcxFvuAdQFRS\n9IZePFUt7yklUtrAd0dbs4EwDRRiWu9b6NVWh4nVeMlVOlotq0hQucfJuXACc3m/\nxNx/lpTzjNyHcg/NOvrb9ZFkahqWQtTrIPVdZ3f3YBEGoKf4oZgtWX/j4Ye63j8w\nuKklzWttI66oNAVNUv1ESRdYql/p5/BVSJaVK4bdkXqYHX2j3PrPd30ICwxz0bGd\n41UdMiKMJhlkhIESsB8bcdRAEaMS2OaFKmBYIQF4RuY3syvFizJDtp/QEYfjy9tT\nXokd3Wzs6dncn/yyfvT0+yCDjYsNAgFvBmfHNBorywxILdtgJHuc9oO2EOeg58VK\nVt4eugECgYEA/wxb29pVamwxF71gKx/msBa5kwxV5N7NhTLdYyHwhQVErQlwn7Dg\nJ8qLfZqmn231yoGpKLZsu2mxdRvpd9nvOiW+ZF+fsrS8SEs5dMEqhojALm8rur+Y\n5M0/Sk/A0lCbSmV+X7vmqaGzyNdgH7tYVIxXjAo4sEYN6GevjUB1JQECgYEAw1wZ\nBhhsIvW9gfbuCdiTGlezUuIO3oxjvSSTNUaGAB7GUqB26toBnXi6oQi5iGu/dCYU\n3CILOkV7kTX//2njOfWLp/kP+5nVKDgHoA/0gL609sgrdgkQ0KdZ3iuurimeqvDm\nU5hpPrNcwz7yPJ/M081ve84pHq3wzVKpi1dMNVMCgYEA4e5JxTTg63hR+MyqTylg\nSmanF2sa/7aa6r6HPRTIop1rG7m8Cco+lyEmdiq0JZDb5fr8JXOMWGylZa9HHwNw\nltrukK3gowbVr1jr2dBv4mNrkvaqDzFAuJZU1XhWwDfliH7l9tpV17jFsUmQ/isQ\ncT0tJIG9e/Fiyphm+8K4wwECgYEAwXbCHUQwSoq7aiokX0HHo624G1tcyE2VNCk1\nUuwNJa9UTV01hqvwL4bwoyqluZCin55ayAk6vzEyBoLIiqLM8IfXDrhaeJpF+jdK\nbdt/EcRKJ53hVFnz+f3QxHDT4wu6YqSAI8bqarprIbuDXkAOMq3eOmfWVtiAgITc\n++2uvZsCgYEAmpN2RfHxO3huEWFoE7LTy9WTv4DDHI+g8PeCUpP2pN/UmczInyQ4\nOlKeNTSxn9AkyYx9PJ8i1TIx6GyFIX4pkJczLEu+XINm82MKSBGuRL1EUvkVddx3\n6clZk5BLDXjmCtCr5DGZ01EbT0wsbsBM1GtoCS4+vUQkJVHb0r6/ZdM=\n-----END RSA PRIVATE KEY-----"
}

func TestErrorf(t *testing.T) {
	errorf("error: %s", fmt.Errorf("hello"))
}

func TestCenter(t *testing.T) {
	s := "in the middleya"
	w := 112 // or whatever

	// centered := fmt.Sprintf("|%[1]*s|", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
	// centered = truncateString(centered, 110)
	centered := centeredString(s, w)
	fmt.Println("s", centered, len(centered))
}

type stringPair struct {
	left  string
	right string
}

func TestPrettify(t *testing.T) {
	// #       TPP     CH0

	compareResults := []stringPair{{"\\VED\\Policy\\Certificates\\Division 3\\TestCerta", "NA"},
		{"\\VED\\Policy\\Certificates\\Division 3\\TestCertb", "NA"},
		{"NA", "/TestCertb_20nov26_DE38"},
		{"NA", "/TestCommonName_20nov25_DE13"},
		{"\\VED\\Policy\\Certificates\\Division 3\\localhost", "NA"},
		{"NA", "/mycertfromvenafi23"},
		{"NA", "/mycredname11"},
		{"\\VED\\Policy\\Certificates\\Division 3\\mycredname11mm", "/mycredname11mm"},
		{"NA", "/mycredname11zb"},
		{"NA", "/mycredname2"}}
	leftLongest := 0
	rightLongest := 0
	for _, item := range compareResults {
		leftLongest = max(leftLongest, len(item.left))
		rightLongest = max(rightLongest, len(item.right))
	}

	header := fmt.Sprintf("%s%s | %s\n", cyan, centeredString("VENAFI", leftLongest), centeredString("CREDHUB", rightLongest))
	fmt.Print(header)
	fmt.Println(strings.Repeat("-", leftLongest+rightLongest+3))

	for _, item := range compareResults {
		// fmt.Printf("%[1]*s | %[1]*s\n", -leftLongest, item.left, -rightLongest, item.right)
		fmt.Printf("%s%[2]*s %s| %s%[6]*s\n", green, -leftLongest, item.left, cyan, green, -rightLongest, item.right)
	}
	// fmt.Printf("%s|%s")
}
