/*
 * Minio Cloud Storage, (C) 2018 Minio, Inc.
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package iam

import (
	"fmt"
	"os"

	"github.com/minio/minio/pkg/auth"
	"github.com/minio/minio/pkg/iam/policy"
	"github.com/minio/minio/pkg/iam/validator"
	xnet "github.com/minio/minio/pkg/net"
)

type iamType string

// IAM types implemented.
const (
	IAMOpenID iamType = "openid"
	IAMMinio  iamType = "minio"

	// Add new IAM types here.
)

type policyType string

// Policy types implemented.
const (
	PolicyOPA   policyType = "opa"
	PolicyMinio policyType = "minio"
)

// IAM implements all definitions to capture multi user
// and external identity providers.
type IAM struct {
	Version  string `json:"version"`
	Identity struct {
		Type   iamType `json:"type"`
		OpenID struct {
			JWT validator.JWTArgs `json:"jwt"`
		} `json:"openid"`
		Minio struct {
			Users map[string]auth.Credentials `json:"users"`
		} `json:"minio"`
	} `json:"identity"`

	Policy struct {
		Type  policyType        `json:"type"`
		OPA   iampolicy.OpaArgs `json:"opa"`
		Minio struct {
			Users map[string]iampolicy.Policy `json:"users"`
		} `json:"minio"`
	} `json:"policy"`
}

// New - initializes IAM config.
func New() (*IAM, error) {
	cfg := &IAM{
		Version: "1",
	}
	cfg.Identity.Minio.Users = make(map[string]auth.Credentials)
	cfg.Policy.Minio.Users = make(map[string]iampolicy.Policy)

	if jwksURL := os.Getenv("MINIO_IAM_JWKS_URL"); jwksURL != "" {
		u, err := xnet.ParseURL(jwksURL)
		if err != nil {
			return cfg, err
		}
		cfg.Identity.Type = IAMOpenID
		cfg.Identity.OpenID.JWT.WebKeyURL = u
	}
	if opaURL := os.Getenv("MINIO_IAM_OPA_URL"); opaURL != "" {
		u, err := xnet.ParseURL(opaURL)
		if err != nil {
			return cfg, err
		}
		cfg.Policy.Type = PolicyOPA
		cfg.Policy.OPA.URL = u
		cfg.Policy.OPA.AuthToken = os.Getenv("MINIO_IAM_OPA_AUTHTOKEN")
	}

	return cfg, nil
}

// Validate - validates all fields of IAM config with proper values.
func (iam *IAM) Validate() error {
	if iam.Identity.Type == "" {
		return fmt.Errorf("IAM identity configuration type cannot be empty, supported values are ['jwt', 'minio']")
	}
	if iam.Policy.Type == "" {
		return fmt.Errorf("IAM policy configuration type cannot be empty, supported values are ['opa', 'minio']")
	}
	return nil
}

// GetAuthValidators - returns ValidatorList which contains
// enabled providers in IAM config.
// A new authentication provider is added like below
// * Add a new provider in pkg/iam/validator package.
func (iam *IAM) GetAuthValidators() *validator.Validators {
	validators := validator.NewValidators()

	if iam.Identity.Type == IAMOpenID {
		if iam.Identity.OpenID.JWT.WebKeyURL != nil {
			validators.Add(validator.NewJWT(iam.Identity.OpenID.JWT))
		}
	}

	return validators
}
