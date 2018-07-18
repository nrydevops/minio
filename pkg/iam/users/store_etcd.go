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

package iamusers

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"
	"sync"
	"time"

	etcd "github.com/coreos/etcd/clientv3"
	"github.com/minio/minio/pkg/auth"
)

// Store - iam users store.
type Store struct {
	sync.RWMutex
	Version    string                      `json:"version"`
	Creds      map[string]auth.Credentials `json:"creds"`
	etcdClient *etcd.Client
}

// NewEtcdStore - Initialize a new credetnials store.
func NewEtcdStore(etcdClient *etcd.Client) (*Store, error) {
	cs := &Store{
		Version:    "1",
		Creds:      make(map[string]auth.Credentials),
		etcdClient: etcdClient,
	}
	if err := loadUsersEtcdConfig(etcdClient, cs); err != nil {
		return nil, err
	}
	go cs.watch()
	return cs, nil

}

func loadUsersEtcdConfig(etcdClient *etcd.Client, cs *Store) error {
	r, err := etcdClient.Get(context.Background(), "users", etcd.WithPrefix())
	if err != nil {
		return err
	}
	if r.Count == 0 {
		return nil
	}
	for _, kv := range r.Kvs {
		var cred auth.Credentials
		decoder := json.NewDecoder(bytes.NewReader(kv.Value))
		decoder.DisallowUnknownFields()
		if err := decoder.Decode(&cred); err != nil {
			return err
		}
		key := strings.TrimPrefix(string(kv.Key), "users/")
		cs.Creds[key] = cred
	}
	return nil
}

func (s *Store) watch() {
	for watchResp := range s.etcdClient.Watch(context.Background(), "users", etcd.WithPrefix()) {
		for _, event := range watchResp.Events {
			key := strings.TrimPrefix(string(event.Kv.Key), "users/")
			if event.Type == etcd.EventTypeDelete {
				s.Lock()
				delete(s.Creds, key)
				s.Unlock()
			}
			if event.IsCreate() || event.IsModify() {
				var cred auth.Credentials
				if err := json.Unmarshal(event.Kv.Value, &cred); err != nil {
					continue
				}
				s.Lock()
				s.Creds[cred.AccessKey] = cred
				s.Unlock()
			}
		}
	}
}

// Set - a new credential.
func (s *Store) Set(cred auth.Credentials) error {
	if !cred.IsValid() {
		return errors.New("invalid credentials")
	}

	data, err := json.Marshal(cred)
	if err != nil {
		return err
	}

	ttl := cred.Expiration.Unix() - time.Now().UTC().Unix()
	if ttl > 0 {
		var resp *etcd.LeaseGrantResponse
		resp, err = s.etcdClient.Grant(context.Background(), ttl)
		if err != nil {
			return err
		}
		_, err = s.etcdClient.Put(context.Background(), fmt.Sprintf("users/%s", cred.AccessKey),
			string(data), etcd.WithLease(resp.ID))
	} else {
		_, err = s.etcdClient.Put(context.Background(), fmt.Sprintf("users/%s", cred.AccessKey),
			string(data))
	}

	return err
}

// Get - get a new user info, access key.
func (s *Store) Get(accessKey string) (auth.Credentials, bool) {
	s.RLock()
	defer s.RUnlock()

	cred, ok := s.Creds[accessKey]
	return cred, cred.IsValid() && ok
}

// Delete - delete a user.
func (s *Store) Delete(accessKey string) {
	s.Lock()
	defer s.Unlock()

	s.etcdClient.Delete(context.Background(), fmt.Sprintf("users/%s", accessKey))
}
