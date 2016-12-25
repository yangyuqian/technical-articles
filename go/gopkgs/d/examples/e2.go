package main

import (
	"database/sql"
	"encoding/json"
	"fmt"
)

type NullBool sql.NullBool

func (f *NullBool) MarshalJSON() ([]byte, error) {
	if f.Valid {
		return json.Marshal(f.Bool)
	} else {
		return json.Marshal(nil)
	}
}

type NullInt64 sql.NullInt64

func (f *NullInt64) MarshalJSON() ([]byte, error) {
	if f.Valid {
		return json.Marshal(f.Int64)
	} else {
		return json.Marshal(nil)
	}
}

func (f *NullInt64) UnmarshalJSON(in []byte) (err error) {
	if string(in) == "null" {
		return
	} else {
		f.Valid = true
		return json.Unmarshal(in, &f.Int64)
	}
}

type NullString sql.NullString

func (f *NullString) MarshalJSON() ([]byte, error) {
	if f.Valid {
		return json.Marshal(f.String)
	} else {
		return json.Marshal(nil)
	}
}

func (f *NullString) UnmarshalJSON(in []byte) (err error) {
	if string(in) == "null" {
		return
	} else {
		f.Valid = true
		return json.Unmarshal(in, &f.String)
	}
}

type NullFloat64 sql.NullFloat64

func (f *NullFloat64) MarshalJSON() ([]byte, error) {
	if f.Valid {
		return json.Marshal(f.Float64)
	} else {
		return json.Marshal(nil)
	}
}

type A struct {
	F1  string          `json:"f1"`
	F21 sql.NullBool    `json:"f21"`
	F22 NullBool        `json:"f22"`
	F23 NullBool        `json:"f23"`
	F31 sql.NullInt64   `json:"f31"`
	F32 NullInt64       `json:"f32"`
	F33 NullInt64       `json:"f33"`
	F41 sql.NullString  `json:"f41"`
	F42 NullString      `json:"f42"`
	F43 NullString      `json:"f43"`
	F51 sql.NullFloat64 `json:"f51"`
	F52 NullFloat64     `json:"f52"`
	F53 NullFloat64     `json:"f53"`
}

type B struct {
	F11 NullInt64  `json:"F11"`
	F12 NullInt64  `json:"F12"`
	F13 NullInt64  `json:"F13"`
	F21 NullString `json:"F21"`
	F22 NullString `json:"F22"`
}

func main() {
	m(&A{
		F1:  "test-f1",
		F22: NullBool{true, true},
		F32: NullInt64{1, true},
		F42: NullString{"valid string", true},
		F52: NullFloat64{1.01, true},
	})

	um(`{"F11": 1, "F12": 1, "F13": null, "F21": null, "F22": "valid string"}`, &B{})
}

func m(in interface{}) {
	if out, err := json.Marshal(in); err == nil {
		fmt.Println(string(out))
	} else {
		panic(err)
	}
}

func um(in string, mod interface{}) {
	if err := json.Unmarshal([]byte(in), mod); err == nil {
		fmt.Printf("%+v\n", mod)
	} else {
		panic(err)
	}
}
