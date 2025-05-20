package koanfuri

import (
	"fmt"

	"github.com/knadh/koanf/maps"
	"github.com/knadh/koanf/providers/confmap"
	"github.com/knadh/koanf/v2"
)

// Merge combines the configuration from another KoanfURI instance into this one.
// The configuration from the other instance will be merged on top of the current configuration.
// Returns an error if either KoanfURI instance is nil or if the merge operation fails.
func (k *KoanfURI) Merge(other *KoanfURI, mergeStrategy string) error {
	// Check for nil instances
	if k == nil {
		return fmt.Errorf("cannot merge into nil KoanfURI")
	}
	if other == nil {
		return fmt.Errorf("cannot merge from nil KoanfURI")
	}

	// Check for nil koanf instances
	if k.konfig == nil {
		return fmt.Errorf("destination KoanfURI has nil konfig")
	}
	if other.konfig == nil {
		return fmt.Errorf("source KoanfURI has nil konfig")
	}

	if mergeStrategy == "preserve" {
		k.konfig.Load(confmap.Provider(other.konfig.Raw(), "."), nil, koanf.WithMergeFunc(customKoanfMergeFuncPreserveSlice))
	} else if mergeStrategy == "overwrite" {
		k.konfig.Load(confmap.Provider(other.konfig.Raw(), "."), nil, koanf.WithMergeFunc(customKoanfMergeFuncNoPreserveSlice))
	} else {
		return fmt.Errorf("invalid merge strategy: %s", mergeStrategy)
	}

	return nil
}

func customKoanfMergeFuncPreserveSlice(src, dest map[string]interface{}) error {
	// First pass: look for and handle "__TOMBSTONE__" values
	for k, v := range src {
		if str, ok := v.(string); ok && str == "__TOMBSTONE__" {
			delete(src, k)  // Remove from source
			delete(dest, k) // Remove from destination
			continue
		}

		// Handle nested maps
		if srcMap, ok := v.(map[string]interface{}); ok {
			if destMap, exists := dest[k].(map[string]interface{}); exists {
				customKoanfMergeFuncPreserveSlice(srcMap, destMap)
				continue
			}
		}

		if srcSlice, ok := v.([]interface{}); ok {
			if destSlice, exists := dest[k].([]interface{}); exists {
				mergedSlice := make([]interface{}, len(destSlice))
				copy(mergedSlice, destSlice)
				mergedSlice = append(mergedSlice, srcSlice...)
				src[k] = mergedSlice
			}
		}
	}
	// Perform regular merge for remaining values
	maps.Merge(src, dest)
	return nil
}

// customKoanfMergeFuncNoPreserveSlice is a custom merge function that handles "__TOMBSTONE__" values and deletes the key if set to
// "__TOMBSTONE__". it calls itself recursively. Otherwise, uses default maps.merge behavior with slices (replace slice wholesale)
func customKoanfMergeFuncNoPreserveSlice(src, dest map[string]interface{}) error {
	// First pass: look for and handle "__TOMBSTONE__" values
	for k, v := range src {
		if str, ok := v.(string); ok && str == "__TOMBSTONE__" {
			delete(src, k)  // Remove from source
			delete(dest, k) // Remove from destination
			continue
		}

		// Handle nested maps
		if srcMap, ok := v.(map[string]interface{}); ok {
			if destMap, exists := dest[k].(map[string]interface{}); exists {
				customKoanfMergeFuncNoPreserveSlice(srcMap, destMap)
				continue
			}
		}

	}
	// Perform regular merge for remaining values
	maps.Merge(src, dest)
	return nil
}
