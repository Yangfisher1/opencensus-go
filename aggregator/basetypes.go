// Copyright 2017, OpenCensus Authors
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

package aggregator

import (
	"time"
)

// Annotation represents a text annotation with a set of attributes and a timestamp.
type Annotation struct {
	Time       time.Time
	Message    string
	Attributes map[string]interface{}
}

// Attribute represents a key-value pair on a span, link or annotation.
// Construct with one of: BoolAttribute, Int64Attribute, or StringAttribute.
type Attribute struct {
	key   string
	value interface{}
}

// Key returns the attribute's key
func (a *Attribute) Key() string {
	return a.key
}

// Value returns the attribute's value
func (a *Attribute) Value() interface{} {
	return a.value
}

// BoolAttribute returns a bool-valued attribute.
func BoolAttribute(key string, value bool) Attribute {
	return Attribute{key: key, value: value}
}

// Int64Attribute returns an int64-valued attribute.
func Int64Attribute(key string, value int64) Attribute {
	return Attribute{key: key, value: value}
}

// Float64Attribute returns a float64-valued attribute.
func Float64Attribute(key string, value float64) Attribute {
	return Attribute{key: key, value: value}
}

// StringAttribute returns a string-valued attribute.
func StringAttribute(key string, value string) Attribute {
	return Attribute{key: key, value: value}
}

// Status is the status of a Span.
type Status struct {
	// Code is a status code.  Zero indicates success.
	//
	// If Code will be propagated to Google APIs, it ideally should be a value from
	// https://github.com/googleapis/googleapis/blob/master/google/rpc/code.proto .
	Code    int32
	Message string
}
