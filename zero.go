// Copyright 2015 Chen Xianren. All rights reserved.
// Use of this source code is governed by a MIT
// license that can be found in the LICENSE file.

package scrud

import (
	"database/sql/driver"
	"fmt"
	"strconv"
	"time"
)

type Bool bool // null as false

func (v *Bool) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Bool) Value() (driver.Value, error) {
	if v {
		return bool(v), nil
	}
	return nil, nil
}

type Int int // null as 0

func (v *Int) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Int) Value() (driver.Value, error) {
	if v != 0 {
		return int64(v), nil
	}
	return nil, nil
}

type Int8 int8 // null as 0

func (v *Int8) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Int8) Value() (driver.Value, error) {
	if v != 0 {
		return int64(v), nil
	}
	return nil, nil
}

type Int16 int16 // null as 0

func (v *Int16) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Int16) Value() (driver.Value, error) {
	if v != 0 {
		return int64(v), nil
	}
	return nil, nil
}

type Int32 int32 // null as 0

func (v *Int32) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Int32) Value() (driver.Value, error) {
	if v != 0 {
		return int64(v), nil
	}
	return nil, nil
}

type Int64 int64 // null as 0

func (v *Int64) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Int64) Value() (driver.Value, error) {
	if v != 0 {
		return int64(v), nil
	}
	return nil, nil
}

type Float32 float32 // null as 0.0

func (v *Float32) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Float32) Value() (driver.Value, error) {
	if v != 0 {
		return float64(v), nil
	}
	return nil, nil
}

type Float64 float64 // null as 0.0

func (v *Float64) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Float64) Value() (driver.Value, error) {
	if v != 0 {
		return float64(v), nil
	}
	return nil, nil
}

type String string // null as empty string

func (v *String) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v String) Value() (driver.Value, error) {
	if v != "" {
		return string(v), nil
	}
	return nil, nil
}

type Time time.Time // null as zero time

func (v *Time) Scan(value interface{}) error {
	return convertAssign(v, value)
}

func (v Time) Value() (driver.Value, error) {
	if t := time.Time(v); !t.IsZero() {
		return t, nil
	}
	return nil, nil
}

func convertAssign(dst, src interface{}) error {
	switch d := dst.(type) {
	case *Bool:
		switch s := src.(type) {
		case bool:
			*d = Bool(s)
			return nil
		case nil:
			*d = false
			return nil
		case string, []byte:
			v, err := strconv.ParseBool(toString(src))
			if err == nil {
				*d = Bool(v)
			}
			return err
		case int64:
			if s == 0 {
				*d = false
				return nil
			} else if s == 1 {
				*d = true
				return nil
			}
		case float64, time.Time:
		}
	case *Int:
		switch src.(type) {
		case int64, float64, []byte, string:
			v, err := strconv.Atoi(toString(src))
			if err == nil {
				*d = Int(v)
			}
			return err
		case nil:
			*d = 0
			return nil
		case bool, time.Time:
		}
	case *Int8:
		switch src.(type) {
		case int64, float64, []byte, string:
			v, err := strconv.ParseInt(toString(src), 10, 8)
			if err == nil {
				*d = Int8(v)
			}
			return err
		case nil:
			*d = 0
			return nil
		case bool, time.Time:
		}
	case *Int16:
		switch src.(type) {
		case int64, float64, []byte, string:
			v, err := strconv.ParseInt(toString(src), 10, 16)
			if err == nil {
				*d = Int16(v)
			}
			return err
		case nil:
			*d = 0
			return nil
		case bool, time.Time:
		}
	case *Int32:
		switch src.(type) {
		case int64, float64, []byte, string:
			v, err := strconv.ParseInt(toString(src), 10, 32)
			if err == nil {
				*d = Int32(v)
			}
			return err
		case nil:
			*d = 0
			return nil
		case bool, time.Time:
		}
	case *Int64:
		switch s := src.(type) {
		case int64:
			*d = Int64(s)
		case nil:
			*d = 0
			return nil
		case float64, []byte, string:
			v, err := strconv.ParseInt(toString(src), 10, 64)
			if err == nil {
				*d = Int64(v)
			}
			return err
		case bool, time.Time:
		}
	case *Float32:
		switch src.(type) {
		case float64, int64, []byte, string:
			v, err := strconv.ParseFloat(toString(src), 32)
			if err == nil {
				*d = Float32(v)
			}
			return err
		case nil:
			*d = 0
			return nil
		case bool, time.Time:
		}
	case *Float64:
		switch s := src.(type) {
		case float64:
			*d = Float64(s)
		case nil:
			*d = 0
			return nil
		case int64, []byte, string:
			v, err := strconv.ParseFloat(toString(src), 64)
			if err == nil {
				*d = Float64(v)
			}
			return err
		case bool, time.Time:
		}
	case *String:
		switch src.(type) {
		case string, []byte, int64, float64, bool, time.Time:
			*d = String(toString(src))
			return nil
		case nil:
			*d = ""
			return nil
		}
	case *Time:
		switch s := src.(type) {
		case time.Time:
			*d = Time(s)
			return nil
		case nil:
			*d = Time{}
			return nil
		case string, []byte, int64, float64, bool:
		}
	}
	return fmt.Errorf("unsupported driver -> Scan pair: %T -> %T", src, dst)
}

func toString(src interface{}) string {
	switch v := src.(type) {
	case string:
		return v
	case []byte:
		return string(v)
	}
	return fmt.Sprintf("%v", src)
}
