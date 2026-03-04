//go:build unit

// Copyright 2024 Alexandre Mahdhaoui
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

package controller

import (
	"context"
	"testing"

	"github.com/alexandremahdhaoui/forge-tracker/internal/types"
	"github.com/alexandremahdhaoui/forge-tracker/internal/util/mocks/mockadapter"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"
)

func TestTrackingSetService_Create_Valid(t *testing.T) {
	store := mockadapter.NewMockTrackingSetStore(t)
	svc := NewTrackingSetService(store)
	ctx := context.Background()

	ts := types.TrackingSet{Name: "my-project"}
	store.EXPECT().Create(mock.Anything, ts).Return(nil)

	result, err := svc.Create(ctx, ts)
	require.NoError(t, err)
	assert.Equal(t, "my-project", result.Name)
}

func TestTrackingSetService_Create_EmptyName(t *testing.T) {
	store := mockadapter.NewMockTrackingSetStore(t)
	svc := NewTrackingSetService(store)
	ctx := context.Background()

	_, err := svc.Create(ctx, types.TrackingSet{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "must not be empty")
}

func TestTrackingSetService_Create_InvalidName(t *testing.T) {
	cases := []struct {
		name string
	}{
		{"has spaces"},
		{"has/slash"},
		{"has.dot"},
		{"has@symbol"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			store := mockadapter.NewMockTrackingSetStore(t)
			svc := NewTrackingSetService(store)
			ctx := context.Background()

			_, err := svc.Create(ctx, types.TrackingSet{Name: tc.name})
			assert.Error(t, err)
			assert.Contains(t, err.Error(), "alphanumeric")
		})
	}
}

func TestTrackingSetService_Create_ValidNames(t *testing.T) {
	cases := []string{"my-project", "my_project", "MyProject123", "a"}
	for _, name := range cases {
		t.Run(name, func(t *testing.T) {
			store := mockadapter.NewMockTrackingSetStore(t)
			svc := NewTrackingSetService(store)
			ctx := context.Background()

			ts := types.TrackingSet{Name: name}
			store.EXPECT().Create(mock.Anything, ts).Return(nil)

			_, err := svc.Create(ctx, ts)
			require.NoError(t, err)
		})
	}
}
