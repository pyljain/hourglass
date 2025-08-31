package hourglass

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/require"
)

func TestConnectAndDisconnect(t *testing.T) {
	h, err := New(&Config{
		RedisAddress:  "localhost:6379",
		RedisPassword: "",
		Limits:        map[string]int{},
	})

	require.Nil(t, err)

	err = h.Close()
	require.Nil(t, err)
}

func TestInvalidConnectShouldThrowAnError(t *testing.T) {
	_, err := New(&Config{
		RedisAddress:  "localhost:6399",
		RedisPassword: "",
		Limits:        map[string]int{},
	})

	require.NotNil(t, err)

}

func TestConsume(t *testing.T) {

	limits := map[string]int{
		"feature1": 5,
		"feature2": 3,
	}

	ctx := context.Background()

	h, err := New(&Config{
		RedisAddress:  "localhost:6379",
		RedisPassword: "",
		Limits:        limits,
	})

	require.Nil(t, err)
	defer h.Close()

	tt := []struct {
		description              string
		featureName              string
		username                 string
		expectedIncrementedValue int
		expectedCanRunFeature    bool
		existingLimits           map[string]int
	}{
		{
			description:              "When the daily threshold is not hit, value should be successfully incremented",
			featureName:              "feature2",
			username:                 "test",
			expectedIncrementedValue: 1,
			existingLimits: map[string]int{
				"feature1": 0,
				"feature2": 0,
			},
			expectedCanRunFeature: true,
		},
		{
			description:              "When the daily threshold is exceeded, value should not be successfully incremented",
			featureName:              "feature2",
			username:                 "test",
			expectedIncrementedValue: 3,
			existingLimits: map[string]int{
				"feature1": 3,
				"feature2": 3,
			},
			expectedCanRunFeature: false,
		},
		{
			description:              "If consume is called for an unregistered feature, it should return -1,-1, true",
			featureName:              "feature-that-does-not-exist",
			username:                 "test",
			expectedIncrementedValue: -1,
			existingLimits: map[string]int{
				"feature1": 3,
				"feature2": 3,
			},
			expectedCanRunFeature: true,
		},
	}

	for _, test := range tt {
		t.Run(test.description, func(t *testing.T) {
			for feature, limit := range test.existingLimits {
				key := getKey(feature, test.username)
				h.redisClient.Set(ctx, key, limit, 1*time.Minute)
			}

			currrent, _, can := h.Consume(ctx, test.featureName, test.username)

			require.Equal(t, test.expectedCanRunFeature, can)
			require.Equal(t, test.expectedIncrementedValue, currrent)
		})
	}

}

func TestGet(t *testing.T) {
	limits := map[string]int{
		"feature1": 5,
		"feature2": 3,
	}

	ctx := context.Background()

	h, err := New(&Config{
		RedisAddress:  "localhost:6379",
		RedisPassword: "",
		Limits:        limits,
	})

	require.Nil(t, err)
	defer h.Close()

	tt := []struct {
		description          string
		featureName          string
		username             string
		expectedCurrentValue int
		expectedLimit        int
		existingLimits       map[string]interface{}
	}{
		{
			description:          "For an existing feature, the current value should be returned",
			featureName:          "feature2",
			username:             "test",
			expectedCurrentValue: 1,
			expectedLimit:        3,
			existingLimits: map[string]interface{}{
				"feature1": 0,
				"feature2": 1,
			},
		},
		{
			description:          "For a non-existing feature, -1 and -1 should be returned as the current value and limit",
			featureName:          "feature-notexistent",
			username:             "test",
			expectedCurrentValue: -1,
			expectedLimit:        -1,
			existingLimits: map[string]interface{}{
				"feature1": 0,
				"feature2": 1,
			},
		},
		{
			description:          "For an existing feature if the counter is not int but float, return error",
			featureName:          "feature1",
			username:             "test",
			expectedCurrentValue: -1,
			expectedLimit:        5,
			existingLimits: map[string]interface{}{
				"feature1": "a",
				"feature2": 1,
			},
		},
	}

	for _, test := range tt {
		t.Run(test.description, func(t *testing.T) {
			for feature, limit := range test.existingLimits {
				key := getKey(feature, test.username)
				h.redisClient.Set(ctx, key, limit, 1*time.Minute)
			}

			currrent, limit := h.Get(ctx, test.featureName, test.username)

			require.Equal(t, test.expectedCurrentValue, currrent)
			require.Equal(t, test.expectedLimit, limit)
		})
	}

}

func TestCredit(t *testing.T) {
	limits := map[string]int{
		"feature1": 5,
		"feature2": 3,
	}

	ctx := context.Background()

	h, err := New(&Config{
		RedisAddress:  "localhost:6379",
		RedisPassword: "",
		Limits:        limits,
	})

	require.Nil(t, err)
	defer h.Close()

	tt := []struct {
		description          string
		featureName          string
		username             string
		expectedCurrentValue int
		expectedLimit        int
		existingLimits       map[string]interface{}
	}{
		{
			description:          "For an existing feature, the current value should be decremented when crediting",
			featureName:          "feature2",
			username:             "test",
			expectedCurrentValue: 2,
			expectedLimit:        3,
			existingLimits: map[string]interface{}{
				"feature1": 0,
				"feature2": 3,
			},
		},
		{
			description:          "For a non-existing feature, the current value should continue to show -1",
			featureName:          "feature-notexistent",
			username:             "test",
			expectedCurrentValue: -1,
			expectedLimit:        -1,
			existingLimits: map[string]interface{}{
				"feature1": 0,
				"feature2": 1,
			},
		},
	}

	for _, test := range tt {
		t.Run(test.description, func(t *testing.T) {
			for feature, limit := range test.existingLimits {
				key := getKey(feature, test.username)
				h.redisClient.Set(ctx, key, limit, 1*time.Minute)
			}

			currrent, limit := h.Credit(ctx, test.featureName, test.username)

			require.Equal(t, test.expectedCurrentValue, currrent)
			require.Equal(t, test.expectedLimit, limit)
		})
	}

}
