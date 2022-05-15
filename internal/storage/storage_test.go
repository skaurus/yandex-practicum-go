package storage

import (
	"testing"
)

const (
	YA     = "https://ya.ru"
	Google = "https://google.com"
)

func Test_memoryStorage_Shorten(t *testing.T) {
	type fields struct {
		counter *int
		store   map[int]string
	}
	type args struct {
		u string
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   int
	}{
		{"can shorten url", fields{Ptr(0), make(map[int]string)}, args{YA}, 1},
		{"can shorten new url", fields{Ptr(1), map[int]string{1: YA}}, args{YA}, 2},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &memoryStorage{
				counter: tt.fields.counter,
				store:   tt.fields.store,
			}
			if got := s.Shorten(tt.args.u); got != tt.want {
				t.Errorf("Shorten(%v, %v) = %v, want %v", s, tt.args.u, got, tt.want)
			}
		})
	}
}

func Test_memoryStorage_Unshorten(t *testing.T) {
	type fields struct {
		counter int
		store   map[int]string
	}
	type args struct {
		id int
	}
	tests := []struct {
		name   string
		fields fields
		args   args
		want   string
		want1  bool
	}{
		{"can unshorten url", fields{2, map[int]string{1: YA, 2: Google}}, args{1}, YA, true},
		{"can unshorten url", fields{2, map[int]string{1: YA, 2: Google}}, args{2}, Google, true},
		{"can't unshorten what is not there", fields{2, map[int]string{1: YA, 2: Google}}, args{3}, "", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			s := &memoryStorage{
				counter: tt.fields.counter,
				store:   tt.fields.store,
			}
			got, got1 := s.Unshorten(tt.args.id)
			if got != tt.want {
				t.Errorf("Unshorten(%v, %v) got = %v, want %v", s, tt.args.id, got, tt.want)
			}
			if got1 != tt.want1 {
				t.Errorf("Unshorten(%v, %v) got1 = %v, want %v", s, tt.args.id, got1, tt.want1)
			}
		})
	}
}
