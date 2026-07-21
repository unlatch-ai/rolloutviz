//go:build js && wasm

package main

import (
	"encoding/json"
	"syscall/js"

	"github.com/TheSnakeFang/rlviz/internal/browsercore"
)

var callbacks []js.Func

func result(value any, err error) map[string]any {
	if err != nil {
		return map[string]any{"ok": false, "error": err.Error()}
	}
	return map[string]any{"ok": true, "value": value}
}

func bytesArgument(value js.Value) []byte {
	data := make([]byte, value.Get("byteLength").Int())
	js.CopyBytesToGo(data, value)
	return data
}

func expose(name string, callback func([]js.Value) map[string]any) {
	fn := js.FuncOf(func(_ js.Value, args []js.Value) any {
		payload, err := json.Marshal(callback(args))
		if err != nil {
			return `{"ok":false,"error":"could not encode result"}`
		}
		return string(payload)
	})
	callbacks = append(callbacks, fn)
	js.Global().Set(name, fn)
}

func main() {
	expose("rlvizParse", func(args []js.Value) map[string]any {
		if len(args) != 2 {
			return result(nil, &argumentError{"rlvizParse expects Uint8Array and filename"})
		}
		payload, err := browsercore.Parse(bytesArgument(args[0]), args[1].String())
		return result(string(payload), err)
	})
	expose("rlvizDecodeAdapter", func(args []js.Value) map[string]any {
		if len(args) != 3 {
			return result(nil, &argumentError{"rlvizDecodeAdapter expects canonical bytes, filename, and source size"})
		}
		payload, err := browsercore.DecodeAdapterResult(bytesArgument(args[0]), args[1].String(), args[2].Int())
		return result(string(payload), err)
	})
	expose("rlvizAnalyze", func(args []js.Value) map[string]any {
		if len(args) != 2 {
			return result(nil, &argumentError{"rlvizAnalyze expects collection JSON and trajectory ID"})
		}
		collection, err := browsercore.DecodeCollection([]byte(args[0].String()))
		if err != nil {
			return result(nil, err)
		}
		return result(browsercore.Analysis(collection, args[1].String()))
	})
	expose("rlvizCompare", func(args []js.Value) map[string]any {
		if len(args) != 3 {
			return result(nil, &argumentError{"rlvizCompare expects collection JSON and two trajectory IDs"})
		}
		collection, err := browsercore.DecodeCollection([]byte(args[0].String()))
		if err != nil {
			return result(nil, err)
		}
		return result(browsercore.Compare(collection, args[1].String(), args[2].String()))
	})
	js.Global().Set("rlvizWasmReady", true)
	select {}
}

type argumentError struct{ message string }

func (err *argumentError) Error() string { return err.message }
