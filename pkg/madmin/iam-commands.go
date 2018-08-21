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
 *
 */

package madmin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/minio/minio/pkg/quick"
)

// GetIAMConfig - retrieves IAM configuration from a Minio server, automatically decrypts
// the incoming encrypted content as well.
func (adm *AdminClient) GetIAMConfig() ([]byte, error) {
	// Execute GET on /minio/admin/v1/iam to get IAM config.
	resp, err := adm.executeMethod("GET",
		requestData{relPath: "/v1/iam"})
	defer closeResponse(resp)
	if err != nil {
		return nil, err
	}

	if resp.StatusCode != http.StatusOK {
		return nil, httpRespToErrorResponse(resp)
	}
	defer resp.Body.Close()

	return DecryptData(adm.secretAccessKey, resp.Body)
}

// SetIAMConfig - set IAM configuration supplied as Reader.
func (adm *AdminClient) SetIAMConfig(config io.Reader) (err error) {
	const maxConfigJSONSize = 1 * 1024 * 1024 // 1MiB

	// Read configuration bytes
	configBuf := make([]byte, maxConfigJSONSize+1)
	n, err := io.ReadFull(config, configBuf)
	if err == nil {
		return bytes.ErrTooLarge
	}
	if err != io.ErrUnexpectedEOF {
		return err
	}
	configBytes := configBuf[:n]

	type configVersion struct {
		Version string `json:"version,omitempty"`
	}
	var cfg configVersion

	// Check if read data is in json format
	if err = json.Unmarshal(configBytes, &cfg); err != nil {
		return errors.New("Invalid JSON format: " + err.Error())
	}

	// Check if the provided json file has "version" key set
	if cfg.Version == "" {
		return errors.New("Missing or unset \"version\" key in json file")
	}
	// Validate there are no duplicate keys in the JSON
	if err = quick.CheckDuplicateKeys(string(configBytes)); err != nil {
		return errors.New("Duplicate key in json file: " + err.Error())
	}

	econfigBytes, err := EncryptData(adm.secretAccessKey, configBytes)
	if err != nil {
		return err
	}

	reqData := requestData{
		relPath: "/v1/iam",
		content: econfigBytes,
	}

	// Execute PUT on /minio/admin/v1/iam to set IAM config.
	resp, err := adm.executeMethod("PUT", reqData)

	defer closeResponse(resp)
	if err != nil {
		return err
	}

	if resp.StatusCode != http.StatusOK {
		return httpRespToErrorResponse(resp)
	}

	return nil
}
