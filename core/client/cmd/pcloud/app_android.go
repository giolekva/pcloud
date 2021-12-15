package main

import (
	"errors"
	"unsafe"

	"gioui.org/app"
	"github.com/giolekva/pcloud/core/client/jni"
)

type androidApp struct {
	jvm      *jni.JVM
	appCtx   jni.Object
	activity jni.Object
}

func createApp() App {
	return &androidApp{
		jvm:    (*jni.JVM)(unsafe.Pointer(app.JavaVM())),
		appCtx: jni.Object(app.AppContext()),
	}
}

func (a *androidApp) LaunchBarcodeScanner() error {
	return jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.activity)
		m := jni.GetMethodID(env, cls, "launchBarcodeScanner", "()Ljava/lang/String;")
		_, err := jni.CallObjectMethod(env, a.activity, m)
		return err
	})
}

func (a *androidApp) OnView(e app.ViewEvent) error {
	a.deleteActivityRef()
	view := jni.Object(e.View)
	if view == 0 {
		return nil
	}
	activity, err := a.contextForView(view)
	if err != nil {
		return err
	}
	a.activity = activity
	return nil
}

func (a *androidApp) deleteActivityRef() {
	if a.activity == 0 {
		return
	}
	jni.Do(a.jvm, func(env *jni.Env) error {
		jni.DeleteGlobalRef(env, a.activity)
		return nil
	})
	a.activity = 0
}

func (a *androidApp) contextForView(view jni.Object) (jni.Object, error) {
	if view == 0 {
		return 0, errors.New("Should not reach")
	}
	var ctx jni.Object
	err := jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, view)
		m := jni.GetMethodID(env, cls, "getContext", "()Landroid/content/Context;")
		var err error
		ctx, err = jni.CallObjectMethod(env, view, m)
		ctx = jni.NewGlobalRef(env, ctx)
		return err
	})
	if err != nil {
		return 0, err
	}
	return ctx, nil
}
