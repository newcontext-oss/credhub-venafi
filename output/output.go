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

package output

import (
	"fmt"
	"log"
	"os"

	"github.com/newcontext-oss/credhub-venafi/config"
)

// Red defines the color red for this app
var Red = "\033[31m"

// Green defines the color green for this app
var Green = "\033[32m"

// Cyan defines the color cyan for this app
var Cyan = "\033[36m"

// CenteredString returns a string centered at the given length
func CenteredString(s string, w int) string {
	centered := fmt.Sprintf("%[1]*s", -w, fmt.Sprintf("%[1]*s", (w+len(s))/2, s))
	return centered
}

// Verbose writes a log entry if the log_level is VERBOSE or higher
func Verbose(format string, a ...interface{}) {
	if config.LogLevel >= config.VERBOSE {
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

// Info writes a log entry if the log_level is INFO or higher
func Info(format string, a ...interface{}) {
	if config.LogLevel >= config.INFO {
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

// Print writes a log entry if the log_level is STATUS or higher
func Print(format string, a ...interface{}) {
	if config.LogLevel >= config.STATUS {
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

// Status writes a log entry if the log_level is STATUS or higher
func Status(format string, a ...interface{}) {
	if config.LogLevel >= config.STATUS {
		fmt.Print(Green)
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

// HelpOutput writes a log entry if the log_level is STATUS or higher
func HelpOutput(format string, a ...interface{}) {
	if config.LogLevel >= config.STATUS {
		fmt.Fprintf(os.Stderr, format, a...)
	}
}

// Errorf writes a log entry if the log_level is ERROR or higher
func Errorf(format string, a ...interface{}) {
	if config.LogLevel >= config.ERROR {
		fmt.Print(Red)
		fmt.Printf(format, a...)
		writeToLogFile(format, a...)
	}
}

func writeToLogFile(format string, a ...interface{}) {
	log.Printf(format, a...)
}
