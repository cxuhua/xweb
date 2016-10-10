// Package validator implements value validations
//
// Copyright 2014 Roberto Teixeira <robteix@robteix.com>
//
// Licensed under the Apache License, Version 2.0 (the "License");
// you may not use this file except in compliance with the License.
// You may obtain a copy of the License at
//
//    http://www.apache.org/licenses/LICENSE-2.0
//
// Unless required by applicable law or agreed to in writing, software
// distributed under the License is distributed on an "AS IS" BASIS,
// WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
// See the License for the specific language governing permissions and
// limitations under the License.

package xweb

import (
	"errors"
	"fmt"
	"reflect"
	"regexp"
	"strconv"
	"strings"
	"unicode"
)

func nonzero(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	valid := true
	switch st.Kind() {
	case reflect.String:
		valid = len(st.String()) != 0
	case reflect.Ptr, reflect.Interface:
		valid = !st.IsNil()
	case reflect.Slice, reflect.Map, reflect.Array:
		valid = st.Len() != 0
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		valid = st.Int() != 0
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		valid = st.Uint() != 0
	case reflect.Float32, reflect.Float64:
		valid = st.Float() != 0
	case reflect.Bool:
		valid = st.Bool()
	case reflect.Invalid:
		valid = false // always invalid
	case reflect.Struct:
		valid = true // always valid since only nil pointers are empty
	default:
		return ErrUnsupported
	}
	if !valid {
		return ErrZeroValue
	}
	return nil
}

// length tests whether a variable's length is equal to a given
// value. For strings it tests the number of characters whereas
// for maps and slices it tests the number of items.
func length(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	valid := true
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = int64(len(st.String())) == p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = int64(st.Len()) == p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Int() == p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Uint() == p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		valid = st.Float() == p
	default:
		return ErrUnsupported
	}
	if !valid {
		return ErrLen
	}
	return nil
}

// min tests whether a variable value is larger or equal to a given
// number. For number types, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func min(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	invalid := false
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(len(st.String())) < p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(st.Len()) < p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Int() < p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Uint() < p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Float() < p
	default:
		return ErrUnsupported
	}
	if invalid {
		return ErrMin
	}
	return nil
}

// max tests whether a variable value is lesser than a given
// value. For numbers, it's a simple lesser-than test; for
// strings it tests the number of characters whereas for maps
// and slices it tests the number of items.
func max(v interface{}, param string) error {
	st := reflect.ValueOf(v)
	var invalid bool
	switch st.Kind() {
	case reflect.String:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(len(st.String())) > p
	case reflect.Slice, reflect.Map, reflect.Array:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = int64(st.Len()) > p
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		p, err := asInt(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Int() > p
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64, reflect.Uintptr:
		p, err := asUint(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Uint() > p
	case reflect.Float32, reflect.Float64:
		p, err := asFloat(param)
		if err != nil {
			return ErrBadParameter
		}
		invalid = st.Float() > p
	default:
		return ErrUnsupported
	}
	if invalid {
		return ErrMax
	}
	return nil
}

// regex is the builtin validation function that checks
// whether the string variable matches a regular expression
func regex(v interface{}, param string) error {
	s, ok := v.(string)
	if !ok {
		return ErrUnsupported
	}

	re, err := regexp.Compile(param)
	if err != nil {
		return ErrBadParameter
	}

	if !re.MatchString(s) {
		return ErrRegexp
	}
	return nil
}

// asInt retuns the parameter as a int64
// or panics if it can't convert
func asInt(param string) (int64, error) {
	i, err := strconv.ParseInt(param, 0, 64)
	if err != nil {
		return 0, ErrBadParameter
	}
	return i, nil
}

// asUint retuns the parameter as a uint64
// or panics if it can't convert
func asUint(param string) (uint64, error) {
	i, err := strconv.ParseUint(param, 0, 64)
	if err != nil {
		return 0, ErrBadParameter
	}
	return i, nil
}

// asFloat retuns the parameter as a float64
// or panics if it can't convert
func asFloat(param string) (float64, error) {
	i, err := strconv.ParseFloat(param, 64)
	if err != nil {
		return 0.0, ErrBadParameter
	}
	return i, nil
}

// TextErr is an error that also implements the TextMarshaller interface for
// serializing out to various plain text encodings. Packages creating their
// own custom errors should use TextErr if they're intending to use serializing
// formats like json, msgpack etc.
type TextErr struct {
	Err error
}

// Error implements the error interface.
func (t TextErr) Error() string {
	return t.Err.Error()
}

// MarshalText implements the TextMarshaller
func (t TextErr) MarshalText() ([]byte, error) {
	return []byte(t.Err.Error()), nil
}

var (
	// ErrZeroValue is the error returned when variable has zero valud
	// and nonzero was specified
	ErrZeroValue = TextErr{errors.New("zero value")}
	// ErrMin is the error returned when variable is less than mininum
	// value specified
	ErrMin = TextErr{errors.New("less than min")}
	// ErrMax is the error returned when variable is more than
	// maximum specified
	ErrMax = TextErr{errors.New("greater than max")}
	// ErrLen is the error returned when length is not equal to
	// param specified
	ErrLen = TextErr{errors.New("invalid length")}
	// ErrRegexp is the error returned when the value does not
	// match the provided regular expression parameter
	ErrRegexp = TextErr{errors.New("regular expression mismatch")}
	// ErrUnsupported is the error error returned when a validation rule
	// is used with an unsupported variable type
	ErrUnsupported = TextErr{errors.New("unsupported type")}
	// ErrBadParameter is the error returned when an invalid parameter
	// is provided to a validation rule (e.g. a string where an int was
	// expected (max=foo,len=bar) or missing a parameter when one is required (len=))
	ErrBadParameter = TextErr{errors.New("bad parameter")}
	// ErrUnknownTag is the error returned when an unknown tag is found
	ErrUnknownTag = TextErr{errors.New("unknown tag")}
	// ErrInvalid is the error returned when variable is invalid
	// (normally a nil pointer)
	ErrInvalid = TextErr{errors.New("invalid value")}
)

// ErrorMap is a map which contains all errors from validating a struct.
type ErrorMap map[string]ErrorArray

// ErrorMap implements the Error interface so we can check error against nil.
// The returned error is if existent the first error which was added to the map.
func (err ErrorMap) Error() string {
	for k, errs := range err {
		if len(errs) > 0 {
			return fmt.Sprintf("%s: %s", k, errs.Error())
		}
	}

	return ""
}

// ErrorArray is a slice of errors returned by the Validate function.
type ErrorArray []error

// ErrorArray implements the Error interface and returns the first error as
// string if existent.
func (err ErrorArray) Error() string {
	if len(err) > 0 {
		return err[0].Error()
	}
	return ""
}

// ValidationFunc is a function that receives the value of a
// field and a parameter used for the respective validation tag.
type ValidationFunc func(v interface{}, param string) error

// Validator implements a validator
type Validator struct {
	// Tag name being used.
	tagName string
	// validationFuncs is a map of ValidationFuncs indexed
	// by their name.
	validationFuncs map[string]ValidationFunc
}

// NewValidator creates a new Validator
func NewValidator() *Validator {
	return &Validator{
		tagName: "validate",
		validationFuncs: map[string]ValidationFunc{
			"nonzero": nonzero,
			"len":     length,
			"min":     min,
			"max":     max,
			"regexp":  regex,
		},
	}
}

// SetTag allows you to change the tag name used in structs
func (mv *Validator) SetTag(tag string) {
	mv.tagName = tag
}

// WithTag creates a new Validator with the new tag name. It is
// useful to chain-call with Validate so we don't change the tag
// name permanently: validator.WithTag("foo").Validate(t)
func (mv *Validator) WithTag(tag string) *Validator {
	v := mv.copy()
	v.SetTag(tag)
	return v
}

// Copy a validator
func (mv *Validator) copy() *Validator {
	return &Validator{
		tagName:         mv.tagName,
		validationFuncs: mv.validationFuncs,
	}
}

// SetValidationFunc sets the function to be used for a given
// validation constraint. Calling this function with nil vf
// is the same as removing the constraint function from the list.
func (mv *Validator) SetValidationFunc(name string, vf ValidationFunc) error {
	if name == "" {
		return errors.New("name cannot be empty")
	}
	if vf == nil {
		delete(mv.validationFuncs, name)
		return nil
	}
	mv.validationFuncs[name] = vf
	return nil
}

func (mv *Validator) getFieldName(f reflect.StructField) string {
	if js := f.Tag.Get("json"); js != "" {
		return strings.Split(js, ",")[0]
	} else if xs := f.Tag.Get("xml"); xs != "" {
		return strings.Split(xs, ",")[0]
	} else if fs := f.Tag.Get("form"); fs != "" {
		return strings.Split(fs, ",")[0]
	} else {
		return f.Name
	}
}

// Validate validates the fields of a struct based
// on 'validator' tags and returns errors found indexed
// by the field name.
func (mv *Validator) Validate(v interface{}) error {
	sv := reflect.ValueOf(v)
	st := reflect.TypeOf(v)
	if sv.Kind() == reflect.Ptr && !sv.IsNil() {
		return mv.Validate(sv.Elem().Interface())
	}
	if sv.Kind() != reflect.Struct {
		return ErrUnsupported
	}
	nfields := sv.NumField()
	m := make(ErrorMap)
	for i := 0; i < nfields; i++ {
		f := sv.Field(i)
		// deal with pointers
		for f.Kind() == reflect.Ptr && !f.IsNil() {
			f = f.Elem()
		}
		tag := st.Field(i).Tag.Get(mv.tagName)
		if tag == "-" {
			continue
		}
		fname := mv.getFieldName(st.Field(i))
		var errs ErrorArray
		if tag != "" {
			err := mv.Valid(f.Interface(), tag)
			if errors, ok := err.(ErrorArray); ok {
				errs = errors
			} else if err != nil {
				errs = ErrorArray{err}
			}
		}
		if f.Kind() == reflect.Struct {
			if !unicode.IsUpper(rune(fname[0])) {
				continue
			}
			e := mv.Validate(f.Interface())
			if e, ok := e.(ErrorMap); ok && len(e) > 0 {
				for j, k := range e {
					m[fname+"."+j] = k
				}
			}
		}
		if len(errs) > 0 {
			m[fname] = errs
		}
	}
	if len(m) > 0 {
		return m
	}
	return nil
}

// Valid validates a value based on the provided
// tags and returns errors found or nil.
func (mv *Validator) Valid(val interface{}, tags string) error {
	if tags == "-" {
		return nil
	}
	v := reflect.ValueOf(val)
	if v.Kind() == reflect.Ptr && !v.IsNil() {
		return mv.Valid(v.Elem().Interface(), tags)
	}
	var err error
	switch v.Kind() {
	case reflect.Invalid:
		err = mv.validateVar(nil, tags)
	default:
		err = mv.validateVar(val, tags)
	}
	return err
}

// validateVar validates one single variable
func (mv *Validator) validateVar(v interface{}, tag string) error {
	tags, err := mv.parseTags(tag)
	if err != nil {
		// unknown tag found, give up.
		return err
	}
	errs := make(ErrorArray, 0, len(tags))
	for _, t := range tags {
		if err := t.Fn(v, t.Param); err != nil {
			errs = append(errs, err)
		}
	}
	if len(errs) > 0 {
		return errs
	}
	return nil
}

// tag represents one of the tag items
type tag struct {
	Name  string         // name of the tag
	Fn    ValidationFunc // validation function to call
	Param string         // parameter to send to the validation function
}

// parseTags parses all individual tags found within a struct tag.
func (mv *Validator) parseTags(t string) ([]tag, error) {
	tl := strings.Split(t, ",")
	tags := make([]tag, 0, len(tl))
	for _, i := range tl {
		tg := tag{}
		v := strings.SplitN(i, "=", 2)
		tg.Name = strings.Trim(v[0], " ")
		if tg.Name == "" {
			return []tag{}, ErrUnknownTag
		}
		if len(v) > 1 {
			tg.Param = strings.Trim(v[1], " ")
		}
		var found bool
		if tg.Fn, found = mv.validationFuncs[tg.Name]; !found {
			return []tag{}, ErrUnknownTag
		}
		tags = append(tags, tg)

	}
	return tags, nil
}
