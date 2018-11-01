package rpc

import (
	"regexp"

	"github.com/asaskevich/govalidator"
)

func init() {
	govalidator.SetFieldsRequiredByDefault(true)

	govalidator.TagMap["t_addr"] = govalidator.Validator(func(str string) bool {
		rx := regexp.MustCompile("^(cx|hx)[0-9a-f]{40}$")
		return rx.MatchString(str)
	})

	govalidator.TagMap["t_addr_score"] = govalidator.Validator(func(str string) bool {
		rx := regexp.MustCompile("^cx[0-9a-f]{40}$")
		return rx.MatchString(str)
	})

	govalidator.TagMap["t_addr_eoa"] = govalidator.Validator(func(str string) bool {
		rx := regexp.MustCompile("^hx[0-9a-f]{40}$")
		return rx.MatchString(str)
	})

	govalidator.TagMap["t_hash"] = govalidator.Validator(func(str string) bool {
		rx := regexp.MustCompile("^0x[0-9a-f]{64}$")
		return rx.MatchString(str)
	})

	govalidator.TagMap["t_hash_v2"] = govalidator.Validator(func(str string) bool {
		rx := regexp.MustCompile("^[0-9a-f]{64}$")
		return rx.MatchString(str)
	})

	govalidator.TagMap["t_int"] = govalidator.Validator(func(str string) bool {
		rx := regexp.MustCompile("^0x[0-9a-f]+$")
		return rx.MatchString(str)
	})

	govalidator.TagMap["t_bin_data"] = govalidator.Validator(func(str string) bool {
		rx := regexp.MustCompile("^0x[0-9a-f]+$")
		return len(str)%2 == 0 && rx.MatchString(str)
	})

	govalidator.TagMap["t_sig"] = govalidator.Validator(func(str string) bool {
		return govalidator.IsBase64(str)
	})

}
