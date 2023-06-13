//go:build unit || !integration

package local_test

import (
	"context"
	"fmt"
	"testing"

	"github.com/bacalhau-project/bacalhau/pkg/objectstore/local"
)

func benchmarkWriteRead(i int, b *testing.B) {
	ctx := context.Background()
	db, _ := local.New(ctx, local.WithPrefixes("test"))
	defer db.Close(ctx)

	type data struct {
		ID string
	}

	counter := 0

	d := data{ID: "1"}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for x := 0; x < i; x++ {
			db.Put(ctx, "test", d.ID, d)
		}
	}

	for n := 0; n < b.N; n++ {
		for x := 0; x < i; x++ {
			_, _ = db.Get(ctx, "test", d.ID, d)
			counter = counter + 1
		}
	}
}

func benchmarkWrite(i int, b *testing.B) {
	ctx := context.Background()
	db, _ := local.New(ctx, local.WithPrefixes("test"))
	defer db.Close(ctx)

	type data struct {
		ID string
	}

	d := data{ID: "1"}

	b.ResetTimer()
	for n := 0; n < b.N; n++ {
		for x := 0; x < i; x++ {
			db.Put(ctx, "test", fmt.Sprintf("%d", x), d)
		}
	}
}

func benchmarkRead(i int, b *testing.B) {
	ctx := context.Background()
	db, _ := local.New(ctx, local.WithPrefixes("test"))
	defer db.Close(ctx)

	type data struct {
		ID string
	}

	d := data{ID: "1"}

	for x := 0; x < i; x++ {
		db.Put(ctx, "test", fmt.Sprintf("%d", x), d)
	}

	b.ResetTimer()

	for n := 0; n < b.N; n++ {
		for x := 0; x < i; x++ {
			_, _ = db.Get(ctx, "test", fmt.Sprintf("%d", x), d)
		}
	}
}

func BenchmarkReadWrite1(b *testing.B)   { benchmarkWriteRead(1, b) }
func BenchmarkReadWrite100(b *testing.B) { benchmarkWriteRead(100, b) }

func BenchmarkWrite1(b *testing.B)   { benchmarkWrite(1, b) }
func BenchmarkWrite100(b *testing.B) { benchmarkWrite(100, b) }

func BenchmarkRead1(b *testing.B)   { benchmarkRead(1, b) }
func BenchmarkRead100(b *testing.B) { benchmarkRead(100, b) }
