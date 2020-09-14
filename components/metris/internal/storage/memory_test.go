package storage

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type FakeObj struct {
	Name string
}

func Test_memoryStorage(t *testing.T) {
	asserts := assert.New(t)
	storage := NewMemoryStorage()

	t.Run("add item", func(t *testing.T) {
		obj := &FakeObj{Name: "item"}
		storage.Put("item", obj)
		data, ok := storage.Get("item")
		asserts.True(ok)
		asserts.Equal(obj, data)
	})

	t.Run("update item", func(t *testing.T) {
		obj := &FakeObj{Name: "item-new"}
		storage.Update("item", obj)
		data, ok := storage.Get("item")
		asserts.True(ok)
		asserts.Equal(obj, data)
	})

	t.Run("delete item", func(t *testing.T) {
		storage.Delete("item")
		_, ok := storage.Get("item")
		asserts.False(ok)
	})

	t.Run("list items", func(t *testing.T) {
		s := &memoryStorage{
			items: map[string]interface{}{
				"item1": &FakeObj{},
				"item2": &FakeObj{},
				"item3": &FakeObj{},
				"item4": &FakeObj{},
			},
		}

		data := s.List()
		asserts.Equal(4, len(data))
	})

	t.Run("list item keys", func(t *testing.T) {
		s := &memoryStorage{
			items: map[string]interface{}{
				"item1": &FakeObj{},
				"item2": &FakeObj{},
				"item3": &FakeObj{},
				"item4": &FakeObj{},
			},
		}

		data := s.ListKeys()
		asserts.ElementsMatch([]string{"item1", "item2", "item3", "item4"}, data)
	})
}
