package models

import "strings"

var vietnameseAccents = map[rune]rune{
	'à': 'a', 'á': 'a', 'ả': 'a', 'ã': 'a', 'ạ': 'a',
	'ă': 'a', 'ằ': 'a', 'ắ': 'a', 'ẳ': 'a', 'ẵ': 'a', 'ặ': 'a',
	'â': 'a', 'ầ': 'a', 'ấ': 'a', 'ẩ': 'a', 'ẫ': 'a', 'ậ': 'a',
	'đ': 'd',
	'è': 'e', 'é': 'e', 'ẻ': 'e', 'ẽ': 'e', 'ẹ': 'e',
	'ê': 'e', 'ề': 'e', 'ế': 'e', 'ể': 'e', 'ễ': 'e', 'ệ': 'e',
	'ì': 'i', 'í': 'i', 'ỉ': 'i', 'ĩ': 'i', 'ị': 'i',
	'ò': 'o', 'ó': 'o', 'ỏ': 'o', 'õ': 'o', 'ọ': 'o',
	'ô': 'o', 'ồ': 'o', 'ố': 'o', 'ổ': 'o', 'ỗ': 'o', 'ộ': 'o',
	'ơ': 'o', 'ờ': 'o', 'ớ': 'o', 'ở': 'o', 'ỡ': 'o', 'ợ': 'o',
	'ù': 'u', 'ú': 'u', 'ủ': 'u', 'ũ': 'u', 'ụ': 'u',
	'ư': 'u', 'ừ': 'u', 'ứ': 'u', 'ử': 'u', 'ữ': 'u', 'ự': 'u',
	'ỳ': 'y', 'ý': 'y', 'ỷ': 'y', 'ỹ': 'y', 'ỵ': 'y',
	'À': 'a', 'Á': 'a', 'Ả': 'a', 'Ã': 'a', 'Ạ': 'a',
	'Ă': 'a', 'Ằ': 'a', 'Ắ': 'a', 'Ẳ': 'a', 'Ẵ': 'a', 'Ặ': 'a',
	'Â': 'a', 'Ầ': 'a', 'Ấ': 'a', 'Ẩ': 'a', 'Ẫ': 'a', 'Ậ': 'a',
	'Đ': 'd',
	'È': 'e', 'É': 'e', 'Ẻ': 'e', 'Ẽ': 'e', 'Ẹ': 'e',
	'Ê': 'e', 'Ề': 'e', 'Ế': 'e', 'Ể': 'e', 'Ễ': 'e', 'Ệ': 'e',
	'Ì': 'i', 'Í': 'i', 'Ỉ': 'i', 'Ĩ': 'i', 'Ị': 'i',
	'Ò': 'o', 'Ó': 'o', 'Ỏ': 'o', 'Õ': 'o', 'Ọ': 'o',
	'Ô': 'o', 'Ồ': 'o', 'Ố': 'o', 'Ổ': 'o', 'Ỗ': 'o', 'Ộ': 'o',
	'Ơ': 'o', 'Ờ': 'o', 'Ớ': 'o', 'Ở': 'o', 'Ỡ': 'o', 'Ợ': 'o',
	'Ù': 'u', 'Ú': 'u', 'Ủ': 'u', 'Ũ': 'u', 'Ụ': 'u',
	'Ư': 'u', 'Ừ': 'u', 'Ứ': 'u', 'Ử': 'u', 'Ữ': 'u', 'Ự': 'u',
	'Ỳ': 'y', 'Ý': 'y', 'Ỷ': 'y', 'Ỹ': 'y', 'Ỵ': 'y',
}

// RemoveAccents converts a string to lowercase and removes Vietnamese diacritics
func RemoveAccents(s string) string {
	s = strings.ToLower(s)
	var sb strings.Builder
	for _, r := range s {
		if val, ok := vietnameseAccents[r]; ok {
			sb.WriteRune(val)
		} else {
			sb.WriteRune(r)
		}
	}
	return sb.String()
}

// Gu inventory helper functions

// FindEquippedGuByID returns the first equipped ACTIVE Gu matching guID, and its index.
func FindEquippedGuByID(inventory GuInventory, guID string) (*GuInstance, int, bool) {
	for i, gu := range inventory {
		if gu.GuID == guID && gu.Equipped && gu.Status == "ACTIVE" {
			return &inventory[i], i, true
		}
	}
	return nil, -1, false
}

// HasEquippedGu checks if player has an equipped ACTIVE Gu of the given type.
func HasEquippedGu(inventory GuInventory, guID string) bool {
	_, _, ok := FindEquippedGuByID(inventory, guID)
	return ok
}

// CountEquippedGuByID counts equipped ACTIVE Gus of the given type.
func CountEquippedGuByID(inventory GuInventory, guID string) int {
	count := 0
	for _, gu := range inventory {
		if gu.GuID == guID && gu.Equipped && gu.Status == "ACTIVE" {
			count++
		}
	}
	return count
}

// CountEquippedActive counts all equipped ACTIVE Gus regardless of type.
func CountEquippedActive(inventory GuInventory) int {
	count := 0
	for _, gu := range inventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			count++
		}
	}
	return count
}

// FindGuByInstanceID finds a Gu by its InstanceID, returning pointer, index, and found flag.
func FindGuByInstanceID(inventory GuInventory, instanceID string) (*GuInstance, int, bool) {
	for i, gu := range inventory {
		if gu.InstanceID == instanceID {
			return &inventory[i], i, true
		}
	}
	return nil, -1, false
}

// FindGuByIDAndRank finds a Gu by GuID and Rank, returning pointer, index, and found flag.
func FindGuByIDAndRank(inventory GuInventory, guID string, rank int) (*GuInstance, int, bool) {
	for i, gu := range inventory {
		if gu.GuID == guID && gu.Rank == rank {
			return &inventory[i], i, true
		}
	}
	return nil, -1, false
}

// FilterEquippedActive returns a new inventory containing only equipped ACTIVE Gus.
func FilterEquippedActive(inventory GuInventory) GuInventory {
	result := make(GuInventory, 0, len(inventory))
	for _, gu := range inventory {
		if gu.Equipped && gu.Status == "ACTIVE" {
			result = append(result, gu)
		}
	}
	return result
}

// RemoveGuByInstanceID removes a Gu by InstanceID from inventory, returning new inventory.
func RemoveGuByInstanceID(inventory GuInventory, instanceID string) GuInventory {
	for i, gu := range inventory {
		if gu.InstanceID == instanceID {
			return append(inventory[:i], inventory[i+1:]...)
		}
	}
	return inventory
}

// RemoveGuByIndex removes a Gu at the given index, returning new inventory.
func RemoveGuByIndex(inventory GuInventory, index int) GuInventory {
	if index < 0 || index >= len(inventory) {
		return inventory
	}
	return append(inventory[:index], inventory[index+1:]...)
}

// FindFirstMatchingGu finds the first Gu matching the predicate.
func FindFirstMatchingGu(inventory GuInventory, fn func(GuInstance) bool) (*GuInstance, int, bool) {
	for i, gu := range inventory {
		if fn(gu) {
			return &inventory[i], i, true
		}
	}
	return nil, -1, false
}

// FindAllMatchingGus returns all Gus matching the predicate.
func FindAllMatchingGus(inventory GuInventory, fn func(GuInstance) bool) []*GuInstance {
	result := make([]*GuInstance, 0)
	for i := range inventory {
		if fn(inventory[i]) {
			result = append(result, &inventory[i])
		}
	}
	return result
}
