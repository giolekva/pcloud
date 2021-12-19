package main

import (
	"bytes"
	"encoding/base64"
	"encoding/json"

	"github.com/giolekva/pcloud/core/client/jni"
)

type androidStorage struct {
	jvm     *jni.JVM
	appCtx  jni.Object
	encrypt jni.MethodID
	decrypt jni.MethodID
}

func CreateStorage(jvm *jni.JVM, appCtx jni.Object) Storage {
	s := &androidStorage{jvm: jvm, appCtx: appCtx}
	jni.Do(jvm, func(env *jni.Env) error {
		appCls := jni.GetObjectClass(env, appCtx)
		s.encrypt = jni.GetMethodID(
			env, appCls,
			"encryptToPref", "(Ljava/lang/String;Ljava/lang/String;)V",
		)
		s.decrypt = jni.GetMethodID(
			env, appCls,
			"decryptFromPref", "(Ljava/lang/String;)Ljava/lang/String;",
		)
		return nil
	})
	return s
}

func (s *androidStorage) Get() (Config, error) {
	var data []byte
	err := jni.Do(s.jvm, func(env *jni.Env) error {
		jfile := jni.JavaString(env, "config")
		plain, err := jni.CallObjectMethod(env, s.appCtx, s.decrypt,
			jni.Value(jfile))
		if err != nil {
			panic(err)
			return err
		}
		b64 := jni.GoString(env, jni.String(plain))
		if b64 == "" {
			return nil
		}
		data, err = base64.RawStdEncoding.DecodeString(b64)
		return err
	})
	var config Config
	if data != nil {
		err = json.NewDecoder(bytes.NewReader(data)).Decode(&config)
	}
	return config, err
}

func (s *androidStorage) Store(config Config) error {
	var data bytes.Buffer
	if err := json.NewEncoder(&data).Encode(config); err != nil {
		return err
	}
	bs64 := base64.RawStdEncoding.EncodeToString(data.Bytes())
	return jni.Do(s.jvm, func(env *jni.Env) error {
		jfile := jni.JavaString(env, "config")
		jplain := jni.JavaString(env, bs64)
		return jni.CallVoidMethod(env, s.appCtx, s.encrypt,
			jni.Value(jfile), jni.Value(jplain))
	})
}
