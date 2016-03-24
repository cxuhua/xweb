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
	. "gopkg.in/check.v1"
	"reflect"
)

type MySuite struct{}

var _ = Suite(&MySuite{})

type Simple struct {
	A int `validate:"min=10"`
}

type TestStruct struct {
	A   int    `validate:"nonzero"`
	B   string `validate:"len=8,min=6,max=4"`
	Sub struct {
		A int `validate:"nonzero"`
		B string
		C float64 `validate:"nonzero,min=1"`
		D *string `validate:"nonzero"`
	}
	D *Simple `validate:"nonzero"`
}

func (ms *MySuite) TestValidate(c *C) {
	t := TestStruct{
		A: 0,
		B: "12345",
	}
	t.Sub.A = 1
	t.Sub.B = ""
	t.Sub.C = 0.0
	t.D = &Simple{10}

	err := Validate(t)
	c.Assert(err, NotNil)

	errs, ok := err.(ErrorMap)
	c.Assert(ok, Equals, true)
	c.Assert(errs["A"], HasError, ErrZeroValue)
	c.Assert(errs["B"], HasError, ErrLen)
	c.Assert(errs["B"], HasError, ErrMin)
	c.Assert(errs["B"], HasError, ErrMax)
	c.Assert(errs["Sub.A"], HasLen, 0)
	c.Assert(errs["Sub.B"], HasLen, 0)
	c.Assert(errs["Sub.C"], HasLen, 2)
	c.Assert(errs["Sub.D"], HasError, ErrZeroValue)
}

func (ms *MySuite) TestValidSlice(c *C) {
	s := make([]int, 0, 10)
	err := Valid(s, "nonzero")
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrZeroValue)

	for i := 0; i < 10; i++ {
		s = append(s, i)
	}

	err = Valid(s, "min=11,max=5,len=9,nonzero")
	c.Assert(err, NotNil)
	errs, ok = err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrMin)
	c.Assert(errs, HasError, ErrMax)
	c.Assert(errs, HasError, ErrLen)
	c.Assert(errs, Not(HasError), ErrZeroValue)
}

func (ms *MySuite) TestValidMap(c *C) {
	m := make(map[string]string)
	err := Valid(m, "nonzero")
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrZeroValue)

	err = Valid(m, "min=1")
	c.Assert(err, NotNil)
	errs, ok = err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrMin)

	m = map[string]string{"A": "a", "B": "a"}
	err = Valid(m, "max=1")
	c.Assert(err, NotNil)
	errs, ok = err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrMax)

	err = Valid(m, "min=2, max=5")
	c.Assert(err, IsNil)

	m = map[string]string{
		"1": "a",
		"2": "b",
		"3": "c",
		"4": "d",
		"5": "e",
	}
	err = Valid(m, "len=4,min=6,max=1,nonzero")
	c.Assert(err, NotNil)
	errs, ok = err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrLen)
	c.Assert(errs, HasError, ErrMin)
	c.Assert(errs, HasError, ErrMax)
	c.Assert(errs, Not(HasError), ErrZeroValue)

}

func (ms *MySuite) TestValidFloat(c *C) {
	err := Valid(12.34, "nonzero")
	c.Assert(err, IsNil)

	err = Valid(0.0, "nonzero")
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrZeroValue)
}

func (ms *MySuite) TestValidInt(c *C) {
	i := 123
	err := Valid(i, "nonzero")
	c.Assert(err, IsNil)

	err = Valid(i, "min=1")
	c.Assert(err, IsNil)

	err = Valid(i, "min=124, max=122")
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrMin)
	c.Assert(errs, HasError, ErrMax)

	err = Valid(i, "max=10")
	c.Assert(err, NotNil)
	errs, ok = err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrMax)
}

func (ms *MySuite) TestValidString(c *C) {
	s := "test1234"
	err := Valid(s, "len=8")
	c.Assert(err, IsNil)

	err = Valid(s, "len=0")
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasError, ErrLen)

	err = Valid(s, "regexp=^[tes]{4}.*")
	c.Assert(err, IsNil)

	err = Valid(s, "regexp=^.*[0-9]{5}$")
	c.Assert(errs, NotNil)

	err = Valid("", "nonzero,len=3,max=1")
	c.Assert(err, NotNil)
	errs, ok = err.(ErrorArray)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasLen, 2)
	c.Assert(errs, HasError, ErrZeroValue)
	c.Assert(errs, HasError, ErrLen)
	c.Assert(errs, Not(HasError), ErrMax)
}

func (ms *MySuite) TestValidateStructVar(c *C) {
	// just verifies that a the given val is a struct
	SetValidationFunc("struct", func(val interface{}, _ string) error {
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Struct {
			return nil
		}
		return ErrUnsupported
	})

	type test struct {
		A int
	}
	err := Valid(test{}, "struct")
	c.Assert(err, IsNil)

	type test2 struct {
		B int
	}
	type test1 struct {
		A test2 `validate:"struct"`
	}

	err = Validate(test1{})
	c.Assert(err, IsNil)

	type test4 struct {
		B int `validate:"foo"`
	}
	type test3 struct {
		A test4
	}
	err = Validate(test3{})
	errs, ok := err.(ErrorMap)
	c.Assert(ok, Equals, true)
	c.Assert(errs["A.B"], HasError, ErrUnknownTag)
}

func (ms *MySuite) TestValidatePointerVar(c *C) {
	// just verifies that a the given val is a struct
	SetValidationFunc("struct", func(val interface{}, _ string) error {
		v := reflect.ValueOf(val)
		if v.Kind() == reflect.Struct {
			return nil
		}
		return ErrUnsupported
	})
	SetValidationFunc("nil", func(val interface{}, _ string) error {
		v := reflect.ValueOf(val)
		if v.IsNil() {
			return nil
		}
		return ErrUnsupported
	})

	type test struct {
		A int
	}
	err := Valid(&test{}, "struct")
	c.Assert(err, IsNil)

	type test2 struct {
		B int
	}
	type test1 struct {
		A *test2 `validate:"struct"`
	}

	err = Validate(&test1{&test2{}})
	c.Assert(err, IsNil)

	type test4 struct {
		B int `validate:"foo"`
	}
	type test3 struct {
		A test4
	}
	err = Validate(&test3{})
	errs, ok := err.(ErrorMap)
	c.Assert(ok, Equals, true)
	c.Assert(errs["A.B"], HasError, ErrUnknownTag)

	err = Valid((*test)(nil), "nil")
	c.Assert(err, IsNil)

	type test5 struct {
		A *test2 `validate:"nil"`
	}
	err = Validate(&test5{})
	c.Assert(err, IsNil)

	type test6 struct {
		A *test2 `validate:"nonzero"`
	}
	err = Validate(&test6{})
	errs, ok = err.(ErrorMap)
	c.Assert(ok, Equals, true)
	c.Assert(errs["A"], HasError, ErrZeroValue)

	err = Validate(&test6{&test2{}})
	c.Assert(err, IsNil)
}

func (ms *MySuite) TestValidateOmittedStructVar(c *C) {
	type test2 struct {
		B int `validate:"min=1"`
	}
	type test1 struct {
		A test2 `validate:"-"`
	}

	t := test1{}
	err := Validate(t)
	c.Assert(err, IsNil)

	errs := Valid(test2{}, "-")
	c.Assert(errs, IsNil)
}

func (ms *MySuite) TestUnknownTag(c *C) {
	type test struct {
		A int `validate:"foo"`
	}
	t := test{}
	err := Validate(t)
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorMap)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasLen, 1)
	c.Assert(errs["A"], HasError, ErrUnknownTag)
}

func (ms *MySuite) TestUnsupported(c *C) {
	type test struct {
		A int     `validate:"regexp=a.*b"`
		B float64 `validate:"regexp=.*"`
	}
	t := test{}
	err := Validate(t)
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorMap)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasLen, 2)
	c.Assert(errs["A"], HasError, ErrUnsupported)
	c.Assert(errs["B"], HasError, ErrUnsupported)
}

func (ms *MySuite) TestBadParameter(c *C) {
	type test struct {
		A string `validate:"min="`
		B string `validate:"len=="`
		C string `validate:"max=foo"`
	}
	t := test{}
	err := Validate(t)
	c.Assert(err, NotNil)
	errs, ok := err.(ErrorMap)
	c.Assert(ok, Equals, true)
	c.Assert(errs, HasLen, 3)
	c.Assert(errs["A"], HasError, ErrBadParameter)
	c.Assert(errs["B"], HasError, ErrBadParameter)
	c.Assert(errs["C"], HasError, ErrBadParameter)
}

type hasErrorChecker struct {
	*CheckerInfo
}

func (c *hasErrorChecker) Check(params []interface{}, names []string) (bool, string) {
	var (
		ok    bool
		slice []error
		value error
	)
	slice, ok = params[0].(ErrorArray)
	if !ok {
		return false, "First parameter is not an Errorarray"
	}
	value, ok = params[1].(error)
	if !ok {
		return false, "Second parameter is not an error"
	}

	for _, v := range slice {
		if v == value {
			return true, ""
		}
	}
	return false, ""
}

func (c *hasErrorChecker) Info() *CheckerInfo {
	return c.CheckerInfo
}

var HasError = &hasErrorChecker{&CheckerInfo{Name: "HasError", Params: []string{"HasError", "expected to contain"}}}
