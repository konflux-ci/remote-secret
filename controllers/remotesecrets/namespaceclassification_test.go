//
// Copyright (c) 2021 Red Hat, Inc.
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

package remotesecrets

import (
	"testing"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
	"github.com/stretchr/testify/assert"
)

func TestClassifyWithNoPriorState(t *testing.T) {
	tests := map[string]*api.RemoteSecret{
		"no-overrides": {
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_b",
					},
				},
			},
		},
		"name-overrides": {
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "sec1",
						},
					},
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "sec2",
						},
					},
				},
			},
		},
		"generateName-overrides": {
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "sec1",
						},
					},
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "sec2",
						},
					},
				},
			},
		},
	}

	for name, rs := range tests {
		t.Run(name, func(t *testing.T) {
			nc := ClassifyTargetNamespaces(rs)

			assert.Len(t, nc.Remove, 0)
			assert.Len(t, nc.Sync, 2)
			assert.Empty(t, nc.DuplicateTargetSpecs)
			assert.Empty(t, nc.OrphanDuplicateStatuses)

			assert.Equal(t, StatusTargetIndex(-1), nc.Sync[SpecTargetIndex(0)])
			assert.Equal(t, StatusTargetIndex(-1), nc.Sync[SpecTargetIndex(1)])
		})
	}
}

func TestClassifyReordered(t *testing.T) {
	tests := map[string]*api.RemoteSecret{
		"no-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_b",
					},
					{
						Namespace: "ns_c",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_b",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
						ServiceAccountNames: []string{},
					},
					{
						Namespace: "ns_c",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
						ServiceAccountNames: []string{},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
						ServiceAccountNames: []string{},
					},
				},
			},
		},
		"no-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_b",
					},
					{
						Namespace: "ns_c",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_b",
						SecretName:          "sec",
						ServiceAccountNames: []string{},
					},
					{
						Namespace:           "ns_c",
						SecretName:          "sec",
						ServiceAccountNames: []string{},
					},
					{
						Namespace:           "ns_a",
						SecretName:          "sec",
						ServiceAccountNames: []string{},
					},
				},
			},
		},
		"name-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "sec1",
						},
					},
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "sec2",
						},
					},
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec2",
						},
						ServiceAccountNames: []string{},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
						ServiceAccountNames: []string{},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec1",
						},
						ServiceAccountNames: []string{},
					},
				},
			},
		},
		"name-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "sec1",
						},
					},
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "sec2",
						},
					},
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_a",
						SecretName:          "sec2",
						ServiceAccountNames: []string{},
					},
					{
						Namespace:           "ns_a",
						SecretName:          "sec",
						ServiceAccountNames: []string{},
					},
					{
						Namespace:           "ns_a",
						SecretName:          "sec1",
						ServiceAccountNames: []string{},
					},
				},
			},
		},
		"generateName-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					GenerateName: "sec-",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "sec1-",
						},
					},
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "sec2-",
						},
					},
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec2-asdf",
						},
						ServiceAccountNames: []string{},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec-asfd",
						},
						ServiceAccountNames: []string{},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec1-asdf",
						},
						ServiceAccountNames: []string{},
					},
				},
			},
		},
		"generateName-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					GenerateName: "sec-",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "sec1-",
						},
					},
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "sec2-",
						},
					},
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_a",
						SecretName:          "sec2-asdf",
						ServiceAccountNames: []string{},
					},
					{
						Namespace:           "ns_a",
						SecretName:          "sec-asfd",
						ServiceAccountNames: []string{},
					},
					{
						Namespace:           "ns_a",
						SecretName:          "sec1-asdf",
						ServiceAccountNames: []string{},
					},
				},
			},
		},
	}

	for name, rs := range tests {
		t.Run(name, func(t *testing.T) {
			nc := ClassifyTargetNamespaces(rs)

			assert.Len(t, nc.Remove, 0)
			assert.Len(t, nc.Sync, 3)
			assert.Empty(t, nc.DuplicateTargetSpecs)
			assert.Empty(t, nc.OrphanDuplicateStatuses)

			assert.Equal(t, StatusTargetIndex(2), nc.Sync[SpecTargetIndex(0)])
			assert.Equal(t, StatusTargetIndex(0), nc.Sync[SpecTargetIndex(1)])
			assert.Equal(t, StatusTargetIndex(1), nc.Sync[SpecTargetIndex(2)])
		})
	}
}

func TestClassifyWithSomeMissingFromStatus(t *testing.T) {
	tests := map[string]*api.RemoteSecret{
		"no-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_b",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_b",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"no-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_b",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_b",
						SecretName:          "sec",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"name-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "seca",
						},
					},
					{
						Namespace: "ns_b",
						Secret: &api.SecretOverride{
							Name: "secb",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_b",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "secb",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"name-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "seca",
						},
					},
					{
						Namespace: "ns_b",
						Secret: &api.SecretOverride{
							Name: "secb",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_b",
						SecretName:          "secb",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"generateName-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					GenerateName: "sec-",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "seca-",
						},
					},
					{
						Namespace: "ns_b",
						Secret: &api.SecretOverride{
							GenerateName: "secb-",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_b",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "secb-asdf",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"generateName-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					GenerateName: "sec-",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "seca-",
						},
					},
					{
						Namespace: "ns_b",
						Secret: &api.SecretOverride{
							GenerateName: "secb-",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_b",
						SecretName:          "secb-asdf",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
	}

	for name, rs := range tests {
		t.Run(name, func(t *testing.T) {
			nc := ClassifyTargetNamespaces(rs)

			assert.Len(t, nc.Remove, 0)
			assert.Len(t, nc.Sync, 2)
			assert.Empty(t, nc.DuplicateTargetSpecs)
			assert.Empty(t, nc.OrphanDuplicateStatuses)

			assert.Equal(t, StatusTargetIndex(-1), nc.Sync[SpecTargetIndex(0)])
			assert.Equal(t, StatusTargetIndex(0), nc.Sync[SpecTargetIndex(1)])
		})
	}
}

func TestClassifyWithSomeMoreInStatus(t *testing.T) {
	tests := map[string]*api.RemoteSecret{
		"no-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_b",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"no-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:  "ns_b",
						SecretName: "sec",
					},
					{
						Namespace:           "ns_a",
						SecretName:          "sec",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"name-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "seca",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_b",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "seca",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"name-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					Name: "sec",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							Name: "seca",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_b",
						SecretName:          "sec",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
					{
						Namespace:           "ns_a",
						SecretName:          "seca",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"generateName-overrides": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					GenerateName: "sec-",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "seca-",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_b",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "sec-asdf",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
					{
						Namespace: "ns_a",
						DeployedSecret: &api.DeployedSecretStatus{
							Name: "seca-asdf",
						},
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
		"generateName-overrides-legacy-secretname": {
			Spec: api.RemoteSecretSpec{
				Secret: api.LinkableSecretSpec{
					GenerateName: "sec-",
				},
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
						Secret: &api.SecretOverride{
							GenerateName: "seca-",
						},
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace:           "ns_b",
						SecretName:          "sec-asdf",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
					{
						Namespace:           "ns_a",
						SecretName:          "seca-asdf",
						ServiceAccountNames: []string{"sa_a", "sa_b"},
					},
				},
			},
		},
	}

	for name, rs := range tests {
		t.Run(name, func(t *testing.T) {
			nc := ClassifyTargetNamespaces(rs)

			assert.Len(t, nc.Remove, 1)
			assert.Len(t, nc.Sync, 1)
			assert.Empty(t, nc.DuplicateTargetSpecs)
			assert.Empty(t, nc.OrphanDuplicateStatuses)

			assert.Equal(t, StatusTargetIndex(1), nc.Sync[SpecTargetIndex(0)])
			assert.Equal(t, StatusTargetIndex(0), nc.Remove[0])
		})
	}
}

func TestClassifyDuplicates(t *testing.T) {
	t.Run("duplicates with matching entries in status", func(t *testing.T) {
		rs := &api.RemoteSecret{
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
				},
			},
		}

		nc := ClassifyTargetNamespaces(rs)

		assert.Empty(t, nc.Remove)
		assert.Len(t, nc.Sync, 1)
		assert.Len(t, nc.DuplicateTargetSpecs, 1)
		assert.Empty(t, nc.OrphanDuplicateStatuses)

		assert.Contains(t, nc.DuplicateTargetSpecs, SpecTargetIndex(0))
		duplicates := nc.DuplicateTargetSpecs[SpecTargetIndex(0)]
		assert.Len(t, duplicates, 2)
		assert.Equal(t, StatusTargetIndex(1), duplicates[SpecTargetIndex(1)])
		assert.Equal(t, StatusTargetIndex(2), duplicates[SpecTargetIndex(2)])
	})

	t.Run("duplicates with unmatched entries in status", func(t *testing.T) {
		rs := &api.RemoteSecret{
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_b",
					},
					{
						Namespace: "ns_a",
					},
				},
			},
		}

		nc := ClassifyTargetNamespaces(rs)

		assert.Len(t, nc.Remove, 1) // the ns_b
		assert.Len(t, nc.Sync, 1)
		assert.Len(t, nc.DuplicateTargetSpecs, 1)
		assert.Empty(t, nc.OrphanDuplicateStatuses)

		assert.Contains(t, nc.DuplicateTargetSpecs, SpecTargetIndex(0))
		duplicates := nc.DuplicateTargetSpecs[SpecTargetIndex(0)]
		assert.Len(t, duplicates, 2)
		assert.Equal(t, StatusTargetIndex(2), duplicates[SpecTargetIndex(1)])
		assert.Equal(t, StatusTargetIndex(-1), duplicates[SpecTargetIndex(2)])
	})

	t.Run("superfluous duplicate entries in status", func(t *testing.T) {
		rs := &api.RemoteSecret{
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
					{
						Namespace: "ns_a",
					},
				},
			},
		}

		nc := ClassifyTargetNamespaces(rs)

		assert.Empty(t, nc.Remove) // the ns_b
		assert.Len(t, nc.Sync, 1)
		assert.Len(t, nc.DuplicateTargetSpecs, 1)
		assert.Len(t, nc.OrphanDuplicateStatuses, 1)

		assert.Contains(t, nc.DuplicateTargetSpecs, SpecTargetIndex(0))
		duplicates := nc.DuplicateTargetSpecs[SpecTargetIndex(0)]
		assert.Len(t, duplicates, 1)
		assert.Equal(t, StatusTargetIndex(1), duplicates[SpecTargetIndex(1)])

		assert.Equal(t, nc.OrphanDuplicateStatuses[0], StatusTargetIndex(2))
	})
}

func TestClassifyByCluster(t *testing.T) {
	t.Run("uses cluster to match spec with status", func(t *testing.T) {
		rs := &api.RemoteSecret{
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						ApiUrl:    "cluster_1",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_2",
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						ApiUrl:    "cluster_2",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_1",
						Namespace: "ns_a",
					},
				},
			},
		}

		nc := ClassifyTargetNamespaces(rs)

		assert.Empty(t, nc.Remove)
		assert.Len(t, nc.Sync, 2)
		assert.Empty(t, nc.DuplicateTargetSpecs)
		assert.Empty(t, nc.OrphanDuplicateStatuses)

		assert.Equal(t, StatusTargetIndex(1), nc.Sync[SpecTargetIndex(0)])
		assert.Equal(t, StatusTargetIndex(0), nc.Sync[SpecTargetIndex(1)])
	})

	t.Run("detects duplicates in clusters separately", func(t *testing.T) {
		rs := &api.RemoteSecret{
			Spec: api.RemoteSecretSpec{
				Targets: []api.RemoteSecretTarget{
					{
						ApiUrl:    "cluster_1",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_2",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_1",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_2",
						Namespace: "ns_a",
					},
				},
			},
			Status: api.RemoteSecretStatus{
				Targets: []api.TargetStatus{
					{
						ApiUrl:    "cluster_2",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_1",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_1",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_2",
						Namespace: "ns_a",
					},
					{
						ApiUrl:    "cluster_2",
						Namespace: "ns_a",
					},
				},
			},
		}

		nc := ClassifyTargetNamespaces(rs)

		assert.Empty(t, nc.Remove)
		assert.Len(t, nc.Sync, 2)
		assert.Len(t, nc.DuplicateTargetSpecs, 2)
		assert.Len(t, nc.OrphanDuplicateStatuses, 1)

		assert.Equal(t, StatusTargetIndex(1), nc.Sync[SpecTargetIndex(0)])
		assert.Equal(t, StatusTargetIndex(0), nc.Sync[SpecTargetIndex(1)])

		duplicates_1 := nc.DuplicateTargetSpecs[SpecTargetIndex(0)]
		assert.Len(t, duplicates_1, 1)
		assert.Equal(t, StatusTargetIndex(2), duplicates_1[SpecTargetIndex(2)])

		duplicates_2 := nc.DuplicateTargetSpecs[SpecTargetIndex(1)]
		assert.Len(t, duplicates_2, 1)
		assert.Equal(t, StatusTargetIndex(3), duplicates_2[SpecTargetIndex(3)])

		assert.Equal(t, StatusTargetIndex(4), nc.OrphanDuplicateStatuses[0])
	})
}

func TestClassificationWithOverrides(t *testing.T) {
	t.Run("simple", func(t *testing.T) {
		t.Run("name overrides spec name", func(t *testing.T) {
			tests := map[string]*api.RemoteSecret{
				"secretname-in-status-target-secret": {
					Spec: api.RemoteSecretSpec{
						Secret: api.LinkableSecretSpec{
							Name: "spec",
						},
						Targets: []api.RemoteSecretTarget{
							{
								Namespace: "ns",
								Secret: &api.SecretOverride{
									Name: "override",
								},
							},
						},
					},
					Status: api.RemoteSecretStatus{
						Targets: []api.TargetStatus{
							{
								Namespace: "ns",
								DeployedSecret: &api.DeployedSecretStatus{
									Name: "override",
								},
							},
						},
					},
				},
				"legacy-secretname": {
					Spec: api.RemoteSecretSpec{
						Secret: api.LinkableSecretSpec{
							Name: "spec",
						},
						Targets: []api.RemoteSecretTarget{
							{
								Namespace: "ns",
								Secret: &api.SecretOverride{
									Name: "override",
								},
							},
						},
					},
					Status: api.RemoteSecretStatus{
						Targets: []api.TargetStatus{
							{
								Namespace:  "ns",
								SecretName: "override",
							},
						},
					},
				},
			}

			for name, rs := range tests {
				t.Run(name, func(t *testing.T) {
					nc := ClassifyTargetNamespaces(rs)

					assert.Equal(t, 1, len(nc.Sync))
					assert.Empty(t, nc.DuplicateTargetSpecs)
					assert.Empty(t, nc.Remove)
					assert.Empty(t, nc.OrphanDuplicateStatuses)
				})
			}
		})
		t.Run("generateName overrides generateName in spec", func(t *testing.T) {
			tests := map[string]*api.RemoteSecret{
				"secretname-in-status-target-secret": {
					Spec: api.RemoteSecretSpec{
						Secret: api.LinkableSecretSpec{
							GenerateName: "spec",
						},
						Targets: []api.RemoteSecretTarget{
							{
								Namespace: "ns",
								Secret: &api.SecretOverride{
									GenerateName: "override",
								},
							},
						},
					},
					Status: api.RemoteSecretStatus{
						Targets: []api.TargetStatus{
							{
								Namespace: "ns",
								DeployedSecret: &api.DeployedSecretStatus{
									Name: "overrideasdf",
								},
							},
						},
					},
				},
				"legacy-secretname": {
					Spec: api.RemoteSecretSpec{
						Secret: api.LinkableSecretSpec{
							GenerateName: "spec",
						},
						Targets: []api.RemoteSecretTarget{
							{
								Namespace: "ns",
								Secret: &api.SecretOverride{
									GenerateName: "override",
								},
							},
						},
					},
					Status: api.RemoteSecretStatus{
						Targets: []api.TargetStatus{
							{
								Namespace:  "ns",
								SecretName: "overrideasdf",
							},
						},
					},
				},
			}

			for name, rs := range tests {
				t.Run(name, func(t *testing.T) {
					nc := ClassifyTargetNamespaces(rs)

					assert.Equal(t, 1, len(nc.Sync))
					assert.Empty(t, nc.DuplicateTargetSpecs)
					assert.Empty(t, nc.Remove)
					assert.Empty(t, nc.OrphanDuplicateStatuses)
				})
			}
		})
	})

	t.Run("concrete name disregards generateName", func(t *testing.T) {
		tests := map[string]*api.RemoteSecret{
			"secretname-in-status-target-secret": {
				Spec: api.RemoteSecretSpec{
					Secret: api.LinkableSecretSpec{
						GenerateName: "spec",
					},
					Targets: []api.RemoteSecretTarget{
						{
							Namespace: "ns",
							Secret: &api.SecretOverride{
								Name: "override",
							},
						},
						{
							Namespace: "ns",
						},
					},
				},
				Status: api.RemoteSecretStatus{
					Targets: []api.TargetStatus{
						{
							Namespace: "ns",
							DeployedSecret: &api.DeployedSecretStatus{
								Name: "overrideasdf",
							},
						},
						{
							Namespace: "ns",
							DeployedSecret: &api.DeployedSecretStatus{
								Name: "override",
							},
						},
						{
							Namespace: "ns",
							DeployedSecret: &api.DeployedSecretStatus{
								Name: "specasdf",
							},
						},
					},
				},
			},
			"legacy-secretname": {
				Spec: api.RemoteSecretSpec{
					Secret: api.LinkableSecretSpec{
						GenerateName: "spec",
					},
					Targets: []api.RemoteSecretTarget{
						{
							Namespace: "ns",
							Secret: &api.SecretOverride{
								Name: "override",
							},
						},
						{
							Namespace: "ns",
						},
					},
				},
				Status: api.RemoteSecretStatus{
					Targets: []api.TargetStatus{
						{
							Namespace:  "ns",
							SecretName: "overrideasdf",
						},
						{
							Namespace:  "ns",
							SecretName: "override",
						},
						{
							Namespace:  "ns",
							SecretName: "specasdf",
						},
					},
				},
			},
		}

		for name, rs := range tests {
			t.Run(name, func(t *testing.T) {
				nc := ClassifyTargetNamespaces(rs)

				assert.Equal(t, 2, len(nc.Sync))
				assert.Equal(t, 1, len(nc.Remove))
				assert.Empty(t, nc.DuplicateTargetSpecs)
				assert.Empty(t, nc.OrphanDuplicateStatuses)

				assert.Equal(t, StatusTargetIndex(1), nc.Sync[SpecTargetIndex(0)])
				assert.Equal(t, StatusTargetIndex(2), nc.Sync[SpecTargetIndex(1)])
				assert.Contains(t, nc.Remove, StatusTargetIndex(0))
			})
		}
	})

	t.Run("names take precedence over generateNames", func(t *testing.T) {
		tests := map[string]*api.RemoteSecret{
			"secretname-in-status-target-secret": {
				Spec: api.RemoteSecretSpec{
					Secret: api.LinkableSecretSpec{
						GenerateName: "spec-",
					},
					Targets: []api.RemoteSecretTarget{
						{
							Namespace: "ns",
						},
						{
							Namespace: "ns",
							Secret: &api.SecretOverride{
								Name: "spec-asdf",
							},
						},
						{
							Namespace: "ns",
							Secret: &api.SecretOverride{
								Name:         "spec2-asdf",
								GenerateName: "spec-",
							},
						},
					},
				},
				Status: api.RemoteSecretStatus{
					Targets: []api.TargetStatus{
						{
							Namespace: "ns",
							DeployedSecret: &api.DeployedSecretStatus{
								Name: "spec-asdf",
							},
						},
						{
							Namespace: "ns",
							DeployedSecret: &api.DeployedSecretStatus{
								Name: "spec-asdfasdf",
							},
						},
					},
				},
			},
			"legacy-secretname": {
				Spec: api.RemoteSecretSpec{
					Secret: api.LinkableSecretSpec{
						GenerateName: "spec-",
					},
					Targets: []api.RemoteSecretTarget{
						{
							Namespace: "ns",
						},
						{
							Namespace: "ns",
							Secret: &api.SecretOverride{
								Name: "spec-asdf",
							},
						},
						{
							Namespace: "ns",
							Secret: &api.SecretOverride{
								Name:         "spec2-asdf",
								GenerateName: "spec-",
							},
						},
					},
				},
				Status: api.RemoteSecretStatus{
					Targets: []api.TargetStatus{
						{
							Namespace:  "ns",
							SecretName: "spec-asdf",
						},
						{
							Namespace:  "ns",
							SecretName: "spec-asdfasdf",
						},
					},
				},
			},
		}

		for name, rs := range tests {
			t.Run(name, func(t *testing.T) {
				nc := ClassifyTargetNamespaces(rs)

				assert.Equal(t, 3, len(nc.Sync))
				assert.Empty(t, nc.Remove)
				assert.Empty(t, nc.DuplicateTargetSpecs)
				assert.Empty(t, nc.OrphanDuplicateStatuses)

				assert.Equal(t, StatusTargetIndex(1), nc.Sync[SpecTargetIndex(0)])
				assert.Equal(t, StatusTargetIndex(0), nc.Sync[SpecTargetIndex(1)])
				assert.Equal(t, StatusTargetIndex(-1), nc.Sync[SpecTargetIndex(2)])
			})
		}
	})
}
