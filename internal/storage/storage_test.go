package storage

import (
	"context"
	"testing"

	"github.com/google/go-cmp/cmp"
	"gotest.tools/v3/assert"
)

const (
	YA     = "https://ya.ru"
	Google = "https://google.com"
)

func Test_memoryStorage_Store(t *testing.T) {
	type args struct {
		url     string
		addedBy string
	}
	store := NewMemoryStorage()
	tests := []struct {
		name         string
		args         args
		wantedValue  int
		wantedStruct memoryStorage
	}{
		{"can shorten url", args{YA, "skaurus"}, 1, memoryStorage{
			IntPtr(1),
			map[int]string{1: YA},
			map[string][]int{"skaurus": {1}},
		}},
		{"can shorten new url", args{YA, "skaurus"}, 2, memoryStorage{
			IntPtr(2),
			map[int]string{1: YA, 2: YA},
			map[string][]int{"skaurus": {1, 2}},
		}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.Store(context.Background(), tt.args.url, tt.args.addedBy)
			assert.NilError(t, err)
			assert.Equal(t, tt.wantedValue, got)
			// AllowUnexported не упоминается в документации пакета gotest.tools,
			// но удалось нагуглить решение по тексту ошибки
			assert.DeepEqual(t, &tt.wantedStruct, store, cmp.AllowUnexported(memoryStorage{}))
		})
	}
}

func Test_memoryStorage_GetByID(t *testing.T) {
	type args struct {
		id int
	}
	store := memoryStorage{
		IntPtr(2),
		map[int]string{1: YA, 2: Google},
		map[string][]int{"skaurus": {1, 2}},
	}
	tests := []struct {
		name string
		args args
		want string
		err  error
	}{
		{"can unshorten url", args{1}, YA, nil},
		{"can unshorten url", args{2}, Google, nil},
		{"can't unshorten what is not there", args{3}, "", ErrNotFound},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, err := store.GetByID(context.Background(), tt.args.id)
			if tt.err == nil {
				assert.NilError(t, err)
			} else {
				assert.ErrorIs(t, err, tt.err)
			}
			assert.Equal(t, tt.want, got)
		})
	}
}
