package main

// JNI implementations of Java native callback methods.

import (
	"fmt"
	"unsafe"

	"github.com/giolekva/pcloud/core/client/jni"
)

// #include <jni.h>
import "C"

//export Java_me_lekva_pcloud_PCloudActivity_qrcodeScanned
func Java_me_lekva_pcloud_PCloudActivity_qrcodeScanned(env *C.JNIEnv, this C.jobject, contents C.jobject) {
	jenv := (*jni.Env)(unsafe.Pointer(env))
	code := jni.GoString(jenv, jni.String(contents))
	fmt.Printf("!!!! QRCODE SCANNED: %s\n", code)
}
