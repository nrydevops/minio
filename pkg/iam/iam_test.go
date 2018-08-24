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
	"errors"
	"os"
	"testing"
)

func TestNewIAM(t *testing.T) {
	im, err := New()
	if err != nil {
		t.Fatal(err)
	}
	if err = im.Validate(); err == nil {
		t.Fatal(errors.New("validation should fail"))
	}

	os.Setenv("MINIO_IAM_JWKS_URL", "https://localhost:9443/oauth2/jwks")
	os.Setenv("MINIO_IAM_OPA_URL", "https://localhost:8181/http/authz")

	im, err = New()
	if err != nil {
		t.Fatal(err)
	}
	if err = im.Validate(); err != nil {
		t.Fatal(err)
	}

	os.Unsetenv("MINIO_IAM_JWKS_URL")
	os.Unsetenv("MINIO_IAM_OPA_URL")
}
