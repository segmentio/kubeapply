package provider

import (
	"sync"
)

// diffCache is a basic in-memory cache to handle storing the diffs for all configs in a change.
//
// If these aren't cached, they're re-computed multiple times per Terraform run, which slows
// things down and can lead to "Provider produced inconsistent final plan" errors.
type diffCache struct {
	sync.Mutex

	values map[string]map[string]interface{}
}

func newDiffCache() *diffCache {
	return &diffCache{
		values: map[string]map[string]interface{}{},
	}
}

func (d *diffCache) get(key string) map[string]interface{} {
	d.Lock()
	defer d.Unlock()

	return d.values[key]
}

func (d *diffCache) set(key string, value map[string]interface{}) {
	d.Lock()
	defer d.Unlock()

	d.values[key] = value
}

func (d *diffCache) del(key string) {
	d.Lock()
	defer d.Unlock()

	delete(d.values, key)
}
