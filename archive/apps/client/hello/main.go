package main

import (
	"fmt"
	"os"
	"syscall"
	"unsafe"

	"github.com/ahmetalpbalkan/go-cursor"
	"github.com/bytecodealliance/wasmtime-go"
)

type WindowSize struct {
	Row    uint16
	Col    uint16
	Xpixel uint16
	Ypixel uint16
}

func check(err error) {
	if err != nil {
		panic(err)
	}
}

func main() {
	store := wasmtime.NewStore(wasmtime.NewEngine())
	linker := wasmtime.NewLinker(store.Engine)
	err := linker.DefineWasi()
	check(err)
	wasiConfig := wasmtime.NewWasiConfig()
	store.SetWasi(wasiConfig)
	err = linker.FuncNew(
		"env",
		"hello",
		wasmtime.NewFuncType(
			[]*wasmtime.ValType{wasmtime.NewValType(wasmtime.KindI32)},
			[]*wasmtime.ValType{}),
		func(c *wasmtime.Caller, args []wasmtime.Val) ([]wasmtime.Val, *wasmtime.Trap) {
			if len(args) != 1 {
				check(fmt.Errorf("%+v", args))
			}
			id := args[0].I32()
			fmt.Printf("Hello %d\n", id)
			return []wasmtime.Val{}, nil
		})
	check(err)
	err = linker.FuncWrap("env", "cursorHide", func() {
		fmt.Print(cursor.Hide())
	})
	check(err)
	err = linker.FuncWrap("env", "cursorShow", func() {
		fmt.Print(cursor.Show())
	})
	check(err)
	err = linker.FuncWrap("env", "cursorSet", func(x, y int32) {
		fmt.Print(cursor.MoveTo(int(x), int(y)))
	})
	check(err)
	err = linker.FuncWrap("env", "clearScreen", func() {
		fmt.Print(cursor.ClearEntireScreen())
	})
	check(err)
	err = linker.FuncWrap("env", "flush", func() {
		// fmt.Print(cursor.Flush())
	})
	check(err)
	err = linker.FuncWrap("env", "draw", func(x, y int32) {
		fmt.Printf("\x1b7\x1b[%d;%df%s\x1b8", x, y, ".")
	})
	check(err)
	err = linker.FuncWrap("env", "getSize", func() (int32, int32, int32, int32) {
		var ws WindowSize
		retCode, _, errno := syscall.Syscall(syscall.SYS_IOCTL,
			uintptr(syscall.Stdin),
			uintptr(syscall.TIOCGWINSZ),
			uintptr(unsafe.Pointer(&ws)))
		if int(retCode) == -1 {
			panic(errno)
		}
		return int32(ws.Row), int32(ws.Col), int32(ws.Xpixel), int32(ws.Ypixel)
	})
	check(err)

	wasm, err := os.ReadFile("call_host/target/wasm32-wasi/debug/call_host.wasm")
	check(err)
	module, err := wasmtime.NewModule(store.Engine, wasm)
	check(err)
	instance, err := linker.Instantiate(store, module)
	check(err)
	run := instance.GetFunc(store, "run")
	if run == nil {
		panic("not a function")
	}
	_, err = run.Call(store)
	check(err)
}
