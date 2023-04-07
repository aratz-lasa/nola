package main

import (
	"context"
	"fmt"
	"io/ioutil"
	"testing"

	// So go mod tidy doesn't clean this up. Its used in testdata, but go mod
	// doesn't look in there.
	_ "github.com/wapc/wapc-guest-tinygo"
	"golang.org/x/exp/slog"

	"github.com/tetratelabs/wazero"
	"github.com/tetratelabs/wazero/imports/wasi_snapshot_preview1"
	wasmer "github.com/wasmerio/wasmer-go/wasmer"
)

var dontOptimizeMe int32

func BenchmarkWASMCallOverhead(b *testing.B) {
	b.Run("wasmer-go", func(b *testing.B) {
		wasmBytes, err := ioutil.ReadFile("simple.wasm")
		if err != nil {
			panic(err)
		}

		engine := wasmer.NewEngine()
		store := wasmer.NewStore(engine)
		module, err := wasmer.NewModule(store, wasmBytes)
		if err != nil {
			panic(err)
		}

		importObject := wasmer.NewImportObject()
		instance, err := wasmer.NewInstance(module, importObject)
		if err != nil {
			panic(err)
		}

		sum, err := instance.Exports.GetFunction("sum")
		if err != nil {
			panic(err)
		}

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			v, err := sum(int32(i), int32(i))
			if err != nil {
				panic(err)
			}
			dontOptimizeMe = v.(int32)
		}
	})

	b.Run("wazero", func(b *testing.B) {
		ctx := context.Background()

		r := wazero.NewRuntime(ctx)
		defer r.Close(ctx)

		wasi_snapshot_preview1.MustInstantiate(ctx, r)

		wasmBytes, err := ioutil.ReadFile("simple.wasm")
		if err != nil {
			panic(err)
		}

		mod, err := r.InstantiateModuleFromBinary(ctx, wasmBytes)
		if err != nil {
			slog.Error(err.Error())
			return
		}

		sum := mod.ExportedFunction("sum")

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			v, err := sum.Call(ctx, uint64(i), uint64(i))
			if err != nil {
				panic(err)
			}
			dontOptimizeMe = int32(v[0])
		}
	})

	b.Run("native", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			v, err := nativeSum(int32(i), int32(i))
			if err != nil {
				panic(err)
			}
			dontOptimizeMe = v.(int32)
		}
	})
}

func TestWASMInstances(t *testing.T) {
	t.Skip()

	numInstances := []int{
		10,
		100,
		1000,
		10_000,
		100_000,
		// 1_000_000,
	}

	t.Run("wasmer-go", func(t *testing.T) {
		for _, n := range numInstances {
			instances := make([]*wasmer.Instance, 0, n)
			t.Run(fmt.Sprintf("%d-instances", n), func(t *testing.T) {
				wasmBytes, err := ioutil.ReadFile("../testdata/tinygo/util/main.wasm")
				if err != nil {
					panic(err)
				}

				engine := wasmer.NewEngine()
				store := wasmer.NewStore(engine)
				module, err := wasmer.NewModule(store, wasmBytes)
				if err != nil {
					panic(err)
				}

				for i := 0; i < n; i++ {
					importObject := wasmer.NewImportObject()
					limits, err := wasmer.NewLimits(1, 20)
					if err != nil {
						panic(err)
					}
					memory := wasmer.NewMemory(store, wasmer.NewMemoryType(limits))
					importObject.Register(
						"env",
						map[string]wasmer.IntoExtern{
							"memory": memory,
						},
					)
					instance, err := wasmer.NewInstance(module, importObject)
					if err != nil {
						panic(err)
					}
					instances = append(instances, instance)
				}
			})
		}
	})

}

func nativeSum(a, b int32) (interface{}, error) {
	return a + b, nil
}
