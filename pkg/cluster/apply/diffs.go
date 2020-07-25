package apply

import (
	"bytes"
	"encoding/json"
	"fmt"
	"sort"
	"time"

	log "github.com/sirupsen/logrus"
)

// KubeJSONToObjects converts the results of "kubectl apply ... -o json" into
// a slice of Kubernetes objects.
func KubeJSONToObjects(contents []byte) ([]TypedKubeObj, error) {
	rootObj := TypedKubeObj{}

	// Find the start of the JSON struct, skipping over any warnings in the output
	startIndex := bytes.Index(contents, []byte("{"))
	if startIndex == -1 {
		return nil, fmt.Errorf(
			"kubectl eesponse does not appear to contain JSON: %s",
			string(contents),
		)
	}

	if err := json.Unmarshal(contents[startIndex:], &rootObj); err != nil {
		return nil, fmt.Errorf(
			"Could not unmarshal kubectl JSON response (err=%+v): %s",
			err,
			string(contents),
		)
	}

	if rootObj.Kind == "" {
		return nil, fmt.Errorf(
			"Did not get a kind from kubectl response root object: %s",
			string(contents),
		)
	}

	objs := []TypedKubeObj{}

	if rootObj.Kind == "List" {
		// kubectl returned a list of objects
		for _, item := range rootObj.Items {
			objs = append(objs, item)
		}
	} else {
		// kubectl just returned a single object
		objs = append(objs, rootObj)
	}

	return objs, nil
}

type objKey struct {
	kind      string
	name      string
	namespace string
}

type objDiff struct {
	index  int
	hasNew bool
	oldObj TypedKubeObj
	newObj TypedKubeObj
}

// ObjsToResults diffs old and new object slices to generate a slice of apply
// results for display to the user.
func ObjsToResults(
	oldObjs []TypedKubeObj,
	newObjs []TypedKubeObj,
) ([]Result, error) {
	results := []Result{}

	changes := map[objKey]objDiff{}

	for o, oldObj := range oldObjs {
		key := objToKey(oldObj)
		changes[key] = objDiff{
			index:  o,
			oldObj: oldObj,
		}
	}

	for _, newObj := range newObjs {
		key := objToKey(newObj)
		value, ok := changes[key]
		if !ok {
			log.Warnf("Object %+v not found in old list", key)
			continue
		}
		value.newObj = newObj
		value.hasNew = true
		changes[key] = value
	}

	for key, diff := range changes {
		var creationTime time.Time
		var err error

		if diff.oldObj.CreationTimestamp != "" {
			creationTime, err = time.Parse(time.RFC3339, diff.oldObj.CreationTimestamp)
			if err != nil {
				log.Warnf(
					"Could not parse creation time (%s): %+v",
					diff.oldObj.CreationTimestamp,
					err,
				)
			}
		}

		result := Result{
			Name:       key.name,
			Namespace:  key.namespace,
			Kind:       key.kind,
			CreatedAt:  creationTime,
			OldVersion: diff.oldObj.ResourceVersion,
			index:      diff.index,
		}

		if diff.hasNew {
			result.NewVersion = diff.newObj.ResourceVersion
		}

		results = append(
			results,
			result,
		)
	}

	sort.Slice(
		results, func(a, b int) bool {
			return results[a].index < results[b].index
		},
	)

	return results, nil
}

func objToKey(obj TypedKubeObj) objKey {
	key := objKey{
		kind:      obj.Kind,
		name:      obj.KubeMetadata.Name,
		namespace: obj.KubeMetadata.Namespace,
	}
	return key
}
