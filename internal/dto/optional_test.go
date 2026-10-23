package dto_test

import (
	"encoding/json"
	"testing"

	"github.com/quoihv-3696/realworld-echo/internal/dto"
)

func TestOptional_Unmarshal(t *testing.T) {
	type body struct {
		Bio dto.Optional[string] `json:"bio"`
	}
	str := func(s string) *string { return &s }
	cases := []struct {
		in    string
		set   bool
		value *string
	}{
		{`{}`, false, nil},
		{`{"bio":null}`, true, nil},
		{`{"bio":""}`, true, str("")},
		{`{"bio":"Xin chào"}`, true, str("Xin chào")},
	}
	for _, tc := range cases {
		var got body
		if err := json.Unmarshal([]byte(tc.in), &got); err != nil {
			t.Errorf("%s: lỗi %v", tc.in, err)
			continue
		}
		valueOK := (got.Bio.Value == nil && tc.value == nil) ||
			(got.Bio.Value != nil && tc.value != nil && *got.Bio.Value == *tc.value)
		if got.Bio.Set != tc.set || !valueOK {
			t.Errorf("%s → Set=%v Value=%s, muốn Set=%v Value=%s", tc.in, got.Bio.Set, show(got.Bio.Value), tc.set, show(tc.value))
		}
	}
}

func TestOptional_WrongTypeIsError(t *testing.T) {
	var got struct {
		Bio dto.Optional[string] `json:"bio"`
	}
	if err := json.Unmarshal([]byte(`{"bio":123}`), &got); err == nil {
		t.Error(`{"bio":123} phải lỗi (số không phải chuỗi): trả nguyên lỗi của json.Unmarshal`)
	}
}

func show(p *string) string {
	if p == nil {
		return "nil"
	}
	return `"` + *p + `"`
}
