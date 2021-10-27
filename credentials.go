// This source file is part of the EdgeDB open source project.
//
// Copyright 2020-present EdgeDB Inc. and the EdgeDB authors.
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

package edgedb

import (
	"encoding/json"
	"errors"
	"fmt"
	"io/ioutil"
	"math"
)

type credentials struct {
	host        OptionalStr
	port        OptionalInt32
	user        string
	database    OptionalStr
	password    OptionalStr
	certData    OptionalBytes
	tlsSecurity OptionalStr
}

func readCredentials(path string) (*credentials, error) {
	var (
		values map[string]interface{}
		creds  *credentials
	)

	data, err := ioutil.ReadFile(path)
	if err != nil {
		goto Failed
	}

	err = json.Unmarshal(data, &values)
	if err != nil {
		goto Failed
	}

	creds, err = validateCredentials(values)
	if err != nil {
		goto Failed
	}

	return creds, nil

Failed:
	msg := fmt.Sprintf("cannot read credentials at %q: %v", path, err)
	return nil, &configurationError{msg: msg}
}

func validateCredentials(data map[string]interface{}) (*credentials, error) {
	result := &credentials{}

	if val, ok := data["port"]; ok {
		port, ok := val.(float64)
		if !ok || port != math.Trunc(port) || port < 1 || port > 65535 {
			return nil, errors.New("invalid `port` value")
		}
		result.port.Set(int32(port))
	}

	if user, ok := data["user"]; ok {
		if result.user, ok = user.(string); !ok {
			return nil, errors.New("`user` must be a string")
		}
	} else {
		return nil, errors.New("`user` key is required")
	}

	if host, ok := data["host"]; ok && host != "" {
		h, ok := host.(string)
		if !ok {
			return nil, errors.New("`host` must be a string")
		}
		result.host.Set(h)
	}

	if database, ok := data["database"]; ok {
		db, ok := database.(string)
		if !ok {
			return nil, errors.New("`database` must be a string")
		}
		result.database.Set(db)
	}

	if password, ok := data["password"]; ok {
		pwd, ok := password.(string)
		if !ok {
			return nil, errors.New("`password` must be a string")
		}
		result.password.Set(pwd)
	}

	if certData, ok := data["tls_cert_data"]; ok {
		str, ok := certData.(string)
		if !ok {
			return nil, errors.New("`tls_cert_data` must be a string")
		}
		result.certData.Set([]byte(str))
	}

	if verifyHostname, ok := data["tls_verify_hostname"]; ok {
		val, ok := verifyHostname.(bool)
		if !ok {
			return nil, errors.New("`tls_verify_hostname` must be a boolean")
		}
		v := "strict"
		if !val {
			v = "no_host_verification"
		}
		result.tlsSecurity.Set(v)
	}

	if tlsSecurity, ok := data["tls_security"]; ok {
		val, ok := tlsSecurity.(string)
		if !ok {
			return nil, errors.New("`tls_security` must be a string")
		}
		result.tlsSecurity.Set(val)
	}

	verify, ok := data["tls_verify_hostname"].(bool)
	if !ok {
		return result, nil
	}

	security, ok := data["tls_security"].(string)
	if !ok {
		return result, nil
	}

	switch {
	case verify && security == "insecure":
		fallthrough
	case verify && security == "no_host_verification":
		fallthrough
	case !verify && security == "strict":
		return nil, fmt.Errorf(
			"values tls_verify_hostname=%v and "+
				"tls_security=%q are incompatible",
			verify, security)
	}

	return result, nil
}
