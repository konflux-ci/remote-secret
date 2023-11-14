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
	"sort"

	api "github.com/redhat-appstudio/remote-secret/api/v1beta1"
)

type SpecTargetIndex int
type StatusTargetIndex int

// NamespaceClassification contains the results of the ClassifyTargetNamespaces function.
type NamespaceClassification struct {
	// Sync contains the target namespaces that should be synced. The keys in the map are indices to
	// the target array and the nested namespace array in the remote secret spec and the values are
	// indices to targets array in the remote secret status. These targets were determined to point
	// to the same namespace in the same cluster and therefore should correspond to the same secret
	// that should be synced according to the remote secret spec.
	Sync map[SpecTargetIndex]StatusTargetIndex
	// Remove is an array of indices in the target array of remote secret status that were determined
	// to longer have a counterpart in the target specs and therefore should be removed from
	// the target namespaces.
	Remove []StatusTargetIndex
	// DuplicateTargetSpecs contains all the duplicate entries in the spec array with the corresponding
	// duplicate entries in the status, if any.
	// The top level key is the index of the non-duplicate target in the spec. The nested key is
	// the index of the duplicate target(s) and the nested values are indices in the status that
	// correspond to the duplicate spec entries or -1 if the spec doesn't have a corresponding
	// status entry.
	DuplicateTargetSpecs map[SpecTargetIndex]map[SpecTargetIndex]StatusTargetIndex
	// OrphanDuplicateStatuses are indices in the status targets that mark a duplicate entry that
	// has no corresponding spec target. These may appear if for example the user removes a previously
	// reconciled duplicate entry from the spec.
	// These status entries can be safely removed from the array without any further action.
	OrphanDuplicateStatuses []StatusTargetIndex
}

// ClassifyTargetNamespaces looks at the targets specified in the remote secret spec and the targets
// in the remote secret status and, based on the namespace they each specify, classifies them
// to two groups - the ones that should be synced and the ones that should be removed from the target
// namespaces. Because this is done based on the namespace names, it is robust against the reordering
// of the targets in either the spec or the status of the remote secret.
// Targets in spec that do not have a corresponding status (i.e. the new targets that have not yet been
// deployed to) have the status index set to -1 in the returned classification's Sync map.
func ClassifyTargetNamespaces(rs *api.RemoteSecret) NamespaceClassification {
	specIndices, duplicateSpecs := specNamespaceIndices(rs)
	statusIndices, duplicateStatuses := statusNamespaceIndices(rs)

	ret := NamespaceClassification{
		Sync:                 map[SpecTargetIndex]StatusTargetIndex{},
		Remove:               []StatusTargetIndex{},
		DuplicateTargetSpecs: map[SpecTargetIndex]map[SpecTargetIndex]StatusTargetIndex{},
	}

	// we need to extract the keys so that we can reorder them to process the targets
	// with specific secret names first so that they take precedence over targets
	// that are matched using just a generateName.
	tks := make([]api.TargetKey, 0, len(specIndices))
	for tk := range specIndices {
		tks = append(tks, tk)
	}

	sort.Slice(tks, func(a, b int) bool {
		if tks[a].SecretName != "" && tks[b].SecretName == "" {
			return true
		}
		return false
	})

	for i := range tks {
		tk := tks[i]
		specIdx := specIndices[tk]
		stIdx, stKey, ok := findInStatus(tk, statusIndices)
		if ok {
			ret.Sync[specIdx] = stIdx
			delete(statusIndices, stKey)
		} else {
			ret.Sync[specIdx] = -1
		}
	}

	for _, stIdx := range statusIndices {
		ret.Remove = append(ret.Remove, stIdx)
	}

	// You may ask the Golang authors why they don't include this function in the standard library.
	// You may also understand that repetition ad nauseam is the go way...
	min := func(a, b int) int {
		if a < b {
			return a
		}
		return b
	}

	for tk, specIdxs := range duplicateSpecs {
		originalSpecIdx := specIndices[tk]

		targetDuplicates, targetDuplicatesPresent := ret.DuplicateTargetSpecs[originalSpecIdx]
		if !targetDuplicatesPresent {
			targetDuplicates = map[SpecTargetIndex]StatusTargetIndex{}
			ret.DuplicateTargetSpecs[originalSpecIdx] = targetDuplicates
		}

		// We can match a duplicate spec with a duplicate status rather blindly because we
		// a duplicate does not have any state (i.e. it doesn't deploy anything into the target cluster).
		// They are merely matched by the target key (apiUrl + namespace).

		stIdxs, duplicateKey, foundDuplicates := findInStatus(tk, duplicateStatuses)
		commonLength := min(len(specIdxs), len(stIdxs))

		// match the specs with all the statuses we can
		for i := 0; i < commonLength; i++ {
			targetDuplicates[specIdxs[i]] = stIdxs[i]
		}
		// if there are any unmatched specs, mark them as such
		for i := commonLength; i < len(specIdxs); i++ {
			targetDuplicates[specIdxs[i]] = -1
		}
		// if there are any unmatched status, proclaim them orphans
		for i := commonLength; i < len(stIdxs); i++ {
			ret.OrphanDuplicateStatuses = append(ret.OrphanDuplicateStatuses, stIdxs[i])
		}

		// clean up, because we'll be going through the statuses once more to see if there aren't
		// any more leftovers. These ones have been processed though..
		if foundDuplicates {
			delete(duplicateStatuses, duplicateKey)
		}
	}

	for _, stIdxs := range duplicateStatuses {
		ret.OrphanDuplicateStatuses = append(ret.OrphanDuplicateStatuses, stIdxs...)
	}

	return ret
}

func findInStatus[T any](tk api.TargetKey, statusMap map[api.TargetKey]T) (T, api.TargetKey, bool) {

	var found T
	var foundKey api.TargetKey
	var isFound bool

	for stk, val := range statusMap {
		switch tk.CorrespondsTo(stk) {
		case api.NameCorrespondence:
			// short-circuit, because name correspondence is the best
			return val, stk, true
		case api.GenerateNameCorrespondence:
			if !isFound {
				found = val
				foundKey = stk
				isFound = true
			}
		}
	}
	return found, foundKey, isFound
}

func specNamespaceIndices(rs *api.RemoteSecret) (classifiedSpec map[api.TargetKey]SpecTargetIndex, duplicateTargets map[api.TargetKey][]SpecTargetIndex) {
	classifiedSpec = make(map[api.TargetKey]SpecTargetIndex, len(rs.Spec.Targets))
	duplicateTargets = map[api.TargetKey][]SpecTargetIndex{}
	for ti := range rs.Spec.Targets {
		key := rs.Spec.Targets[ti].ToTargetKey(rs)
		if _, isDuplicate := classifiedSpec[key]; isDuplicate {
			duplicates, duplicatesAlreadyPresent := duplicateTargets[key]
			if !duplicatesAlreadyPresent {
				duplicates = []SpecTargetIndex{SpecTargetIndex(ti)}
			} else {
				duplicates = append(duplicates, SpecTargetIndex(ti))
			}
			duplicateTargets[key] = duplicates
		} else {
			classifiedSpec[key] = SpecTargetIndex(ti)
		}
	}
	return
}

func statusNamespaceIndices(rs *api.RemoteSecret) (classifiedStatus map[api.TargetKey]StatusTargetIndex, duplicateStatuses map[api.TargetKey][]StatusTargetIndex) {
	classifiedStatus = make(map[api.TargetKey]StatusTargetIndex, len(rs.Status.Targets))
	duplicateStatuses = map[api.TargetKey][]StatusTargetIndex{}
	for i := range rs.Status.Targets {
		key := rs.Status.Targets[i].ToTargetKey()
		if _, isDuplicate := classifiedStatus[key]; isDuplicate {
			duplicates, duplicatesAlreadyPresent := duplicateStatuses[key]
			if !duplicatesAlreadyPresent {
				duplicates = []StatusTargetIndex{StatusTargetIndex(i)}
			} else {
				duplicates = append(duplicates, StatusTargetIndex(i))
			}
			duplicateStatuses[key] = duplicates
		} else {
			classifiedStatus[key] = StatusTargetIndex(i)
		}
	}
	return
}
