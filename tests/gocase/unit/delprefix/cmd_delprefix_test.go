/*
 * Licensed to the Apache Software Foundation (ASF) under one
 * or more contributor license agreements.  See the NOTICE file
 * distributed with this work for additional information
 * regarding copyright ownership.  The ASF licenses this file
 * to you under the Apache License, Version 2.0 (the
 * "License"); you may not use this file except in compliance
 * with the License.  You may obtain a copy of the License at
 *
 *   http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing,
 * software distributed under the License is distributed on an
 * "AS IS" BASIS, WITHOUT WARRANTIES OR CONDITIONS OF ANY
 * KIND, either express or implied.  See the License for the
 * specific language governing permissions and limitations
 * under the License.
 *
 */

package deleteprefix

import (
	"context"
	"fmt"
	"testing"
	"time"

	"github.com/apache/kvrocks/tests/gocase/util"
	"github.com/redis/go-redis/v9"
	"github.com/stretchr/testify/require"
)

var instance *util.KvrocksServer

func setup(t *testing.T) *redis.Client {
	instance = util.StartServer(t, map[string]string{})

	// Initialize client with authentication if needed
	client := instance.NewClientWithOption(&redis.Options{
		Addr: instance.HostPort(),
	})
	t.Log("Starting reconnection attempt")
	require.Eventually(t, func() bool {
		err := client.Ping(context.Background()).Err()
		return err == nil || err.Error() == "NOAUTH Authentication required."
	}, 2*time.Minute, time.Second) // Increased timeout to 2 minutes

	return client
}

func teardown() {
	if instance != nil {
		instance.Close()
	}
}

func TestDelPrefix(t *testing.T) {
	client := setup(t)
	defer teardown()

	t.Run("DELPREFIX_ALL", func(t *testing.T) {
		for i := 0; i < 100; i++ {
			require.NoError(t, client.Set(context.Background(), fmt.Sprintf("test:key%d", i), "value", 0).Err())
		}
		require.NoError(t, client.Set(context.Background(), "other:key", "value", 0).Err())

		result, err := client.Do(context.Background(), "DELPREFIX", "test:").Result()
		require.NoError(t, err)
		t.Logf("DELPREFIX result: %v", result)

		for i := 0; i < 100; i++ {
			require.Error(t, client.Get(context.Background(), fmt.Sprintf("test:key%d", i)).Err())
		}
		require.Equal(t, "value", client.Get(context.Background(), "other:key").Val())
	})

	t.Run("Namespace handling", func(t *testing.T) {
		require.NoError(t, client.Set(context.Background(), "{ns}:key1", "value", 0).Err())
		require.NoError(t, client.Set(context.Background(), "{ns}:key2", "value", 0).Err())
		require.NoError(t, client.Set(context.Background(), "other:key", "value", 0).Err())

		result, err := client.Do(context.Background(), "DELPREFIX", "{ns}:").Result()
		require.NoError(t, err)
		t.Logf("DELPREFIX result: %v", result)

		require.Error(t, client.Get(context.Background(), "{ns}:key1").Err())
		require.Error(t, client.Get(context.Background(), "{ns}:key2").Err())
		require.Equal(t, "value", client.Get(context.Background(), "other:key").Val())
	})

	t.Run("Empty prefix", func(t *testing.T) {
		_, err := client.Do(context.Background(), "DELPREFIX", "").Result()
		require.Error(t, err)
		require.Contains(t, err.Error(), "Prefix cannot be empty")
	})

	t.Run("Missing arguments", func(t *testing.T) {
		_, err := client.Do(context.Background(), "DELPREFIX").Result()
		require.Error(t, err)
		require.Contains(t, err.Error(), "wrong number of arguments")
	})
}
