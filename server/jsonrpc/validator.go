package jsonrpc

import (
	"reflect"
	"regexp"

	"gopkg.in/go-playground/validator.v9"
)

var (
	eoaAddressRegex   = regexp.MustCompile("^hx[0-9a-f]{40}$")
	scoreAddressRegex = regexp.MustCompile("^cx[0-9a-f]{40}$")
	hexInt            = regexp.MustCompile("^0x(0|[1-9a-f][0-9a-f]*)$")
	hashRegex         = regexp.MustCompile("^0x[0-9a-f]{64}$")
	rosettaHashRegex  = regexp.MustCompile("^[0b]x[0-9a-f]{64}$")
)

type Validator struct {
	validator *validator.Validate
}

func NewValidator() *Validator {
	v := &Validator{
		validator: validator.New(),
	}

	v.RegisterAlias("optional", "omitempty")

	v.RegisterValidation("version", isJsonRpcVersion)
	v.RegisterValidation("id", isValidIdType)

	v.RegisterValidation("t_addr_eoa", isEoaAddress)
	v.RegisterValidation("t_addr_score", isScoreAddress)
	v.RegisterValidation("t_int", isHexInt)
	v.RegisterValidation("t_hash", isHash)
	v.RegisterValidation("t_rhash", isRosettaHash)

	v.RegisterAlias("t_sig", "base64")
	v.RegisterAlias("t_addr", "t_addr_eoa|t_addr_score")

	return v
}

func (v *Validator) Validate(i interface{}) error {
	return v.validator.Struct(i)
}

func (v *Validator) RegisterValidation(tag string, fn validator.Func) {
	_ = v.validator.RegisterValidation(tag, fn)
}

func (v *Validator) RegisterStructValidation(fn validator.StructLevelFunc, types ...interface{}) {
	v.validator.RegisterStructValidation(fn, types...)
}

func (v *Validator) RegisterAlias(alias string, tags string) {
	v.validator.RegisterAlias(alias, tags)
}

func isJsonRpcVersion(fl validator.FieldLevel) bool {
	return fl.Field().String() == Version
}

func isValidIdType(fl validator.FieldLevel) bool {
	k := fl.Field().Kind()
	return k != reflect.Bool && k != reflect.Array && k != reflect.Map
}

func isEoaAddress(fl validator.FieldLevel) bool {
	return eoaAddressRegex.MatchString(fl.Field().String())
}

func isScoreAddress(fl validator.FieldLevel) bool {
	return scoreAddressRegex.MatchString(fl.Field().String())
}

func isHexInt(fl validator.FieldLevel) bool {
	return hexInt.MatchString(fl.Field().String())
}

func isHash(fl validator.FieldLevel) bool {
	return hashRegex.MatchString(fl.Field().String())
}

func isRosettaHash(fl validator.FieldLevel) bool {
	return rosettaHashRegex.MatchString(fl.Field().String())
}
