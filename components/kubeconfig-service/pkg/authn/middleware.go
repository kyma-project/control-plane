package authn

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"math/rand"
	"net/http"
	"reflect"
	"regexp"
	"strings"
	"time"

	log "github.com/sirupsen/logrus"
	"k8s.io/apiserver/pkg/authentication/authenticator"
)

const NO_GROUPS_IN_TOKEN = "No [groups] in oidc token"
const NO_EXP_IN_TOKEN = "No [exp] in oidc token"
const DECODE_TOKEN_FAILD = "Decode token failed"
const UNMARSHAL_TOKEN_FAILED = "Unmarshal token failed"
const MALFORMED_TOKEN = "Malformed Token"
const saRegex = "[^a-z0-9.-]+"
const saCharset = "abcdefghijklmnopqrstuvwxyz0123456789"

var L2L3OperatiorRoles = []string{"runtimeAdmin", "runtimeOperator"}

type UserInfo struct {
	ID   string
	Role string
	Exp  time.Time
}

func AuthMiddleware(a authenticator.Request) func(http.Handler) http.Handler {
	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {

			userInfo, errMsg, code := ValidateToken(r)
			if errMsg != "" {
				http.Error(w, errMsg, code)
				return
			}

			ctx := context.WithValue(r.Context(), "userInfo", userInfo)
			_, ok, err := a.AuthenticateRequest(r) //Strips "Authorization" Header value on auth success!
			if err != nil {
				log.Errorf("Unable to authenticate the request due to an error: %v", err)
			}
			if !ok || err != nil {
				http.Error(w, "Unauthorized", http.StatusUnauthorized)
				return
			}

			next.ServeHTTP(w, r.WithContext(ctx))
		})
	}
}

func ValidateToken(r *http.Request) (UserInfo, string, int) {
	var userInfo = UserInfo{ID: "", Role: "", Exp: time.Time{}}
	authHeader := strings.Split(r.Header.Get("Authorization"), "Bearer ")
	errMsg, code := "", http.StatusOK

	if len(authHeader) != 2 {
		errMsg, code = MALFORMED_TOKEN, http.StatusBadRequest
	} else {
		var dat map[string]interface{}
		dat, errMsg, code = parseToken(authHeader[1])
		if dat == nil || reflect.ValueOf(dat).IsNil() {
			return userInfo, errMsg, code
		}

		role, errMsg, code := extractRole(dat)
		if errMsg != "" {
			return userInfo, errMsg, code
		}

		userInfo.Role = role
		userInfo.ID = extractUserID(dat)
		userInfo.Exp = extractExpiredData(dat)
	}
	return userInfo, errMsg, code
}

func extractUserID(dat map[string]interface{}) string {
	var userID string
	if dat["login_name"] != nil {
		rawLoginName := fmt.Sprintf("%s", dat["login_name"])
		reg, err := regexp.Compile(saRegex)
		if err == nil {
			userID = reg.ReplaceAllString(strings.ToLower(rawLoginName), "")
			return userID
		}
	}
	userID = StringWithCharset(7, saCharset)
	return userID
}

func StringWithCharset(length int, charset string) string {
	var seededRand *rand.Rand = rand.New(
		rand.NewSource(time.Now().UnixNano()))

	b := make([]byte, length)
	for i := range b {
		b[i] = charset[seededRand.Intn(len(charset))]
	}
	return string(b)
}

func dataAsString(t interface{}) string {
	var data string
	switch reflect.TypeOf(t).Kind() {
	case reflect.Slice:
		strSlice, ok := t.([]string)
		if ok {
			data = strings.Join(strSlice, ", ")
		} else {
			var buffer bytes.Buffer
			s := reflect.ValueOf(t)
			for i := 0; i < s.Len(); i++ {
				if buffer.Len() > 0 {
					buffer.WriteRune(',')
				}
				buffer.WriteString(fmt.Sprintf("%v", s.Index(i).Interface()))
			}
			data = buffer.String()
		}
		fmt.Println(data)
		return data

	default:
	}
	return ""
}

func extractExpiredData(dat map[string]interface{}) time.Time {
	var tm time.Time
	if dat["exp"] == nil {
		return tm
	}
	switch iat := dat["exp"].(type) {
	case float64:
		tm = time.Unix(int64(iat), 0)
	default:
	}
	return tm
}

func extractRole(dat map[string]interface{}) (string, string, int) {
	var role = ""
	if dat["groups"] == nil {
		return role, NO_GROUPS_IN_TOKEN, http.StatusForbidden
	}

	data := dataAsString(dat["groups"])
	if strings.Contains(data, L2L3OperatiorRoles[0]) {
		role = L2L3OperatiorRoles[0]
	} else if strings.Contains(data, L2L3OperatiorRoles[1]) {
		role = L2L3OperatiorRoles[1]
	} else {
		return role, fmt.Sprintf("Not found %s in oidc token", L2L3OperatiorRoles), http.StatusForbidden
	}
	return role, "", http.StatusOK
}

func parseToken(jwtToken string) (map[string]interface{}, string, int) {
	errMsg, code := "", http.StatusOK
	var dat map[string]interface{}
	payload, err := decodePayloadAsRawJSON(jwtToken)
	if err != nil {
		errMsg, code = DECODE_TOKEN_FAILD, http.StatusBadRequest
		return dat, errMsg, code
	}

	if err := json.Unmarshal(payload, &dat); err != nil {
		errMsg, code = UNMARSHAL_TOKEN_FAILED, http.StatusBadRequest
	}
	return dat, errMsg, code
}

// decodePayloadAsRawJSON extracts the payload and returns the raw JSON.
func decodePayloadAsRawJSON(s string) ([]byte, error) {
	parts := strings.SplitN(s, ".", 3)
	if len(parts) != 3 {
		return nil, fmt.Errorf("wants %d segments but got %d segments", 3, len(parts))
	}
	payloadJSON, err := decodePayload(parts[1])
	if err != nil {
		return nil, fmt.Errorf("could not decode the payload: %w", err)
	}
	return payloadJSON, nil
}

func decodePayload(payload string) ([]byte, error) {
	b, err := base64.URLEncoding.WithPadding(base64.NoPadding).DecodeString(payload)
	if err != nil {
		return nil, fmt.Errorf("invalid base64: %w", err)
	}
	return b, nil
}
