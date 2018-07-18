package madmin

import (
	"bytes"
	"encoding/json"
	"errors"
	"io"
	"net/http"

	"github.com/minio/minio/pkg/quick"
)

// GetIAM - returns the config.json of a minio setup, incoming data is encrypted.
func (adm *AdminClient) GetIAM() ([]byte, error) {
	// Execute GET on /minio/admin/v1/config to get config of a setup.
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

	return DecryptServerConfigData(adm.secretAccessKey, resp.Body)
}

// SetIAM - set config supplied as config.json for the setup.
func (adm *AdminClient) SetIAM(config io.Reader) (err error) {
	const maxConfigJSONSize = 256 * 1024 // 256KiB

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

	econfigBytes, err := EncryptServerConfigData(adm.secretAccessKey, configBytes)
	if err != nil {
		return err
	}

	reqData := requestData{
		relPath: "/v1/iam",
		content: econfigBytes,
	}

	// Execute PUT on /minio/admin/v1/config to set config.
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
