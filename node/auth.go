package node

import (
	"encoding/hex"
	"encoding/json"
	"fmt"
	"net/http"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/labstack/echo/v4"

	"github.com/icon-project/goloop/common"
	"github.com/icon-project/goloop/common/crypto"
	"github.com/icon-project/goloop/common/errors"
	"github.com/icon-project/goloop/common/log"
)

const (
	AuthScheme = "goloop"
)

type Auth struct {
	skips map[string]map[string]bool
	users map[string]int64
	addrs map[string]string
	filePath string
	prefix string
	SkipIfEmptyUsers bool
	mtx   sync.Mutex
}

func (a *Auth) MiddlewareFunc() echo.MiddlewareFunc {
	//middleware.KeyAuth response http.StatusBadRequest when 'missing key' error
	//return middleware.KeyAuthWithConfig(middleware.KeyAuthConfig{
	//	Skipper:    a.skipper,
	//	//KeyLookup:  "header:" + echo.HeaderAuthorization, //middleware.DefaultKeyAuthConfig.KeyLookup
	//	AuthScheme: AuthScheme,
	//	Validator:  a.validator,
	//})
	return func(next echo.HandlerFunc) echo.HandlerFunc {
		return func(ctx echo.Context) error {
			if a.skipper(ctx) {
				return next(ctx)
			}
			key, err := a.extractor(ctx)
			if err != nil {
				return echo.NewHTTPError(http.StatusUnauthorized, err.Error())
			}
			valid, err := a.validator(key, ctx)
			if err != nil {
				return err
			} else if valid {
				return next(ctx)
			}
			return echo.ErrUnauthorized
		}
	}
}

func (a *Auth) SetSkip(r *echo.Route, skip bool) {
	m, ok := a.skips[r.Method]
	if !ok {
		m = make(map[string]bool)
		a.skips[r.Method] = m
	}
	m[r.Path] = skip
}

func (a *Auth) skipper(ctx echo.Context) bool {
	if a.SkipIfEmptyUsers && a.IsEmptyUsers() {
		return true
	}
	method := ctx.Request().Method
	if m, ok := a.skips[method]; ok {
		if skip, has := m[ctx.Path()]; has {
			return skip
		}
	}
	return method == http.MethodGet
}

func (a *Auth) extractor(ctx echo.Context) (string, error) {
	auth := ctx.Request().Header.Get(echo.HeaderAuthorization)
	if auth == "" {
		return "", errors.New("missing key in request header")
	}
	l := len(AuthScheme)
	if len(auth) > l+1 && auth[:l] == AuthScheme {
		return auth[l+1:], nil
	}
	return "", errors.New("invalid key in the request header")
}

func parse(s string) map[string]string {
	m := make(map[string]string)
	if s != "" {
		kvs := strings.Split(s, ",")
		for _, kv := range kvs {
			if kv != "" {
				idx := strings.Index(kv, "=")
				if idx > 0 {
					m[kv[:idx]] = kv[(idx + 1):]
				} else {
					m[kv] = ""
				}
			}
		}
	}
	return m
}

func (a *Auth) validator(s string, ctx echo.Context) (b bool, err error) {
	log.Traceln("validator:", s)
	m := parse(s)
	var timestamp int64
	if timestamp, err = strconv.ParseInt(m["Timestamp"], 0, 64); err != nil{
		return
	}
	var signature []byte
	if signature, err = hex.DecodeString(m["Signature"]); err != nil {
		return
	}
	var sig *crypto.Signature
	if sig, err = crypto.ParseSignature(signature); err != nil {
		return
	}
	url := strings.Replace(ctx.Request().URL.EscapedPath(),a.prefix,"",1)
	serialized := fmt.Sprintf("Method=%s,Url=%s,Timestamp=%s",
		ctx.Request().Method, url, m["Timestamp"])

	var pubKey *crypto.PublicKey
	if pubKey, err = sig.RecoverPublicKey(crypto.SHA3Sum256([]byte(serialized))); err != nil {
		return
	}
	addr := common.NewAccountAddressFromPublicKey(pubKey).String()
	log.Traceln("addr:",addr,"serialized:", serialized)

	a.mtx.Lock()
	defer a.mtx.Unlock()
	if id, ok := a.addrs[addr]; ok {
		if ts := a.users[id]; ts < timestamp {
			a.users[id] = timestamp
			log.Traceln("valid signature", ts, timestamp)
			return true, nil
		}
		log.Traceln("old signature", a.users[id], timestamp)
		return false, nil
	}
	log.Traceln("not found user", addr)
	return false, nil
}

func (a *Auth) AddUser(id string) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if _, ok := a.users[id]; ok {
		return errors.Wrapf(ErrAlreadyExists, "User(id=%s) already exists", id)
	}

	addr := &common.Address{}
	if err := addr.SetString(id); err != nil {
		return errors.Wrap(err, "invalid address format")
	}

	if _, ok := a.addrs[addr.String()]; ok {
		return errors.Wrapf(ErrAlreadyExists, "User(addr=%s) already exists", addr.String())
	}

	a.users[id] = time.Now().Unix()
	a.addrs[addr.String()] = id
	if err := a._export(); err != nil {
		panic(err)
	}
	return nil
}

func (a *Auth) RemoveUser(id string) error {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	if _, ok := a.users[id]; !ok {
		return errors.Wrapf(ErrNotExists, "User(id=%s) not exists", id)
	}

	delete(a.users, id)
	var addr string
	for k, v := range a.addrs {
		if v == id {
			addr = k
			break
		}
	}
	delete(a.addrs, addr)
	if err := a._export(); err != nil {
		panic(err)
	}
	return nil
}

func (a *Auth) _users() []string {
	users := make([]string, 0)
	for user := range a.users {
		users = append(users, user)
	}
	return users
}

func (a *Auth) IsEmptyUsers() bool {
	a.mtx.Lock()
	defer a.mtx.Unlock()
	return len(a.users) == 0
}

func (a *Auth) GetUsers() []string {
	a.mtx.Lock()
	defer a.mtx.Unlock()

	return a._users()
}

func (a *Auth) _export() error {
	if a.filePath != "" {
		users := a._users()
		if b, err := json.Marshal(users); err != nil {
			return err
		}else {
			if err = os.WriteFile(a.filePath, b, 0644); err != nil {
				return err
			}
		}
	}
	return nil
}

func NewAuth(filePath, prefix string) *Auth {
	a := &Auth{
		skips: make(map[string]map[string]bool),
		users: make(map[string]int64),
		addrs: make(map[string]string),
		filePath: filePath,
		prefix: prefix,
	}
	if a.filePath != "" {
		if _, err := os.Stat(filePath); err != nil {
			if os.IsNotExist(err) {
				if err = a._export(); err != nil {
					panic(err)
				}
			}
		}
		if b, err := os.ReadFile(filePath); err != nil {
			panic(err)
		} else {
			var users []string
			if err = json.Unmarshal(b, &users); err != nil {
				panic(err)
			}
			for _, user := range users {
				if err = a.AddUser(user); err != nil {
					panic(err)
				}
			}
		}
	}
	return a
}
