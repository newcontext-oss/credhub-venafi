package main

import (
	"fmt"
	"log"
	"os"
	"strings"

	"code.cloudfoundry.org/credhub-cli/credhub/credentials"
	"github.com/Venafi/vcert/pkg/certificate"
)

var red = "\033[31m"
var green = "\033[32m"
var cyan = "\033[36m"

func printCertsPretty(ct ComparisonStrategy, data []CertCompareData) {
	pp, ok := ct.(prettyPrinter)
	if !ok {
		return
	}

	header2 := ""
	headers := pp.headers()
	header0 := headers[0]
	header1 := headers[1]
	if len(headers) > 2 {
		header2 = headers[2]
	}

	leftLongest := 0
	rightLongest := 0
	auxLongest := 0
	for _, d := range data {
		values := pp.values(d.Left, d.Right)
		left := values[0]
		right := values[1]
		if len(headers) > 2 {
			auxLongest = max(auxLongest, len(values[2]))
		}
		leftLongest = max(leftLongest, len(left))
		rightLongest = max(rightLongest, len(right))
	}

	header := ""
	if len(headers) > 2 {
		header = fmt.Sprintf("%s%s | %s | %s\n", cyan, centeredString(header0, leftLongest), centeredString(header1, rightLongest), centeredString(header2, auxLongest))
	} else {
		header = fmt.Sprintf("%s%s | %s\n", cyan, centeredString(header0, leftLongest), centeredString(header1, rightLongest))
	}
	output("%s", header)
	output("%s\n", strings.Repeat("-", leftLongest+rightLongest+auxLongest+3*(len(headers)-1)))

	for _, d := range data {
		values := pp.values(d.Left, d.Right)
		left := values[0]
		right := values[1]
		leftColor := red
		rightColor := red
		if left != "" && right != "" {
			leftColor = green
			rightColor = green
		}

		if len(headers) > 2 {
			output("%s%[2]*s %s| %s%[6]*s %s| %[9]*s\n", leftColor, -leftLongest, left, cyan, rightColor, -rightLongest, right, cyan, auxLongest, values[2])
		} else {
			output("%s%[2]*s %s| %s%[6]*s\n", leftColor, -leftLongest, left, cyan, rightColor, -rightLongest, right)
		}
	}
}

func centeredString(s string, w int) string {
	centered := fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
	return centered
}

func max(x, y int) int {
	if x < y {
		return y
	}
	return x
}

type prettyPrinter interface {
	headers() []string
	values(l *certificate.CertificateInfo, r *credentials.CertificateMetadata) []string
}

func verbose(format string, a ...interface{}) {
	if logLevel >= VERBOSE {
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

func info(format string, a ...interface{}) {
	if logLevel >= INFO {
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

func output(format string, a ...interface{}) {
	if logLevel >= STATUS {
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

func status(format string, a ...interface{}) {
	if logLevel >= STATUS {
		fmt.Print(green)
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

func helpoutput(format string, a ...interface{}) {
	if logLevel >= STATUS {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

func errorf(format string, a ...interface{}) {
	if logLevel >= ERROR {
		fmt.Print(red)
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

func writeToLogFile(format string, a ...interface{}) {
	log.Printf(format, a...)
}
