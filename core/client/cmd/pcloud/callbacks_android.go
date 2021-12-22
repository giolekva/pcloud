package main

// JNI implementations of Java native callback methods.

import (
	"unsafe"

	"github.com/giolekva/pcloud/core/client/jni"
)

// #include <jni.h>
import "C"

//export Java_me_lekva_pcloud_PCloudActivity_qrcodeScanned
func Java_me_lekva_pcloud_PCloudActivity_qrcodeScanned(env *C.JNIEnv, this C.jobject, contents C.jobject) {
	jenv := (*jni.Env)(unsafe.Pointer(env))
	code := jni.GoString(jenv, jni.String(contents))
	p.QRCodeScanned([]byte(code))
}

//export Java_me_lekva_pcloud_PCloudVPNService_connect
func Java_me_lekva_pcloud_PCloudVPNService_connect(env *C.JNIEnv, this C.jobject) {
	jenv := (*jni.Env)(unsafe.Pointer(env))
	p.ConnectRequested(jni.NewGlobalRef(jenv, jni.Object(this)))
}

//export Java_me_lekva_pcloud_PCloudVPNService_disconnect
func Java_me_lekva_pcloud_PCloudVPNService_disconnect(env *C.JNIEnv, this C.jobject) {
	jenv := (*jni.Env)(unsafe.Pointer(env))
	p.DisconnectRequested(jni.NewGlobalRef(jenv, jni.Object(this)))
}
