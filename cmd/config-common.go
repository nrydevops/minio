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

package cmd

import (
	"bytes"
	"context"
	"errors"
	"time"

	"github.com/minio/minio/cmd/logger"
	"github.com/minio/minio/pkg/hash"
)

var errConfigNotFound = errors.New("config file not found")

func readConfig(ctx context.Context, objAPI ObjectLayer, configFile string) (*bytes.Buffer, error) {
	var buffer bytes.Buffer
	// Read entire content by setting size to -1
	if err := objAPI.GetObject(ctx, minioMetaBucket, configFile, 0, -1, &buffer, ""); err != nil {
		// Treat object not found and quorum errors as config not found.
		if isErrObjectNotFound(err) || isErrIncompleteBody(err) || isInsufficientReadQuorum(err) {
			return nil, errConfigNotFound
		}

		logger.GetReqInfo(ctx).AppendTags("configFile", configFile)
		logger.LogIf(ctx, err)
		return nil, err
	}

	// Return config not found on empty content.
	if buffer.Len() == 0 {
		return nil, errConfigNotFound
	}

	return &buffer, nil
}

func saveConfig(objAPI ObjectLayer, configFile string, data []byte) error {
	hashReader, err := hash.NewReader(bytes.NewReader(data), int64(len(data)), "", getSHA256Hash(data))
	if err != nil {
		return err
	}

	_, err = objAPI.PutObject(context.Background(), minioMetaBucket, configFile, hashReader, nil)
	return err
}

func readConfigEtcd(configFile string) ([]byte, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	resp, err := globalEtcdClient.Get(ctx, configFile)
	defer cancel()
	if err != nil {
		return nil, err
	}
	if resp.Count == 0 {
		return nil, errConfigNotFound
	}
	for _, ev := range resp.Kvs {
		if string(ev.Key) == configFile {
			return ev.Value, nil
		}
	}
	return nil, errConfigNotFound
}

// watchConfig - watches for changes on `configFile` on etcd and loads them.
func watchConfig(objAPI ObjectLayer, configFile string, loadCfgFn func(ObjectLayer) error) {
	if globalEtcdClient != nil {
		for watchResp := range globalEtcdClient.Watch(context.Background(), configFile) {
			for _, event := range watchResp.Events {
				if event.IsModify() || event.IsCreate() {
					loadCfgFn(objAPI)
				}
			}
		}
	}
}

func checkConfigEtcd(configFile string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Minute)
	resp, err := globalEtcdClient.Get(ctx, configFile)
	defer cancel()
	if err != nil {
		return err
	}
	if resp.Count == 0 {
		return errConfigNotFound
	}
	return nil
}

func checkConfig(ctx context.Context, configFile string, objAPI ObjectLayer) error {
	if globalEtcdClient != nil {
		return checkConfigEtcd(configFile)
	}

	if _, err := objAPI.GetObjectInfo(ctx, minioMetaBucket, configFile); err != nil {
		// Treat object not found and quorum errors as config not found.
		if isErrObjectNotFound(err) || isInsufficientReadQuorum(err) {
			return errConfigNotFound
		}

		logger.GetReqInfo(ctx).AppendTags("configFile", configFile)
		logger.LogIf(ctx, err)
		return err
	}
	return nil
}
