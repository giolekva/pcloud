package main

import (
	"errors"
	"fmt"
	"unsafe"

	"gioui.org/app"
	"github.com/giolekva/pcloud/core/client/jni"
	"github.com/sirupsen/logrus"
	"github.com/slackhq/nebula"
	"github.com/slackhq/nebula/cert"
	nc "github.com/slackhq/nebula/config"
)

type androidApp struct {
	jvm          *jni.JVM
	appCtx       jni.Object // PCloudApp
	activity     jni.Object // PCloudActivity
	service      jni.Object // PCloudVPNService
	nebulaConfig []byte
	ctrl         *nebula.Control
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

func (a *androidApp) StartVPN(config []byte) error {
	a.nebulaConfig = config
	return jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.activity)
		m := jni.GetMethodID(env, cls, "startVpn", "(Ljava/lang/String;)Ljava/lang/String;")
		jConfig := jni.JavaString(env, string(config))
		_, err := jni.CallObjectMethod(env, a.activity, m, jni.Value(jConfig))
		return err

	})
}

func (a *androidApp) Connect(serv interface{}) error {
	s, ok := serv.(jni.Object)
	if !ok {
		return fmt.Errorf("Unexpected service type: %T", serv)
	}
	jni.Do(a.jvm, func(env *jni.Env) error {
		if jni.IsSameObject(env, s, a.service) {
			// We already have a reference.
			jni.DeleteGlobalRef(env, s)
			return nil
		}
		if a.service != 0 {
			jni.DeleteGlobalRef(env, a.service)
		}
		// netns.SetAndroidProtectFunc(func(fd int) error {
		// 	return jni.Do(a.jvm, func(env *jni.Env) error {
		// 		// Call https://developer.android.com/reference/android/net/VpnService#protect(int)
		// 		// to mark fd as a socket that should bypass the VPN and use the underlying network.
		// 		cls := jni.GetObjectClass(env, s)
		// 		m := jni.GetMethodID(env, cls, "protect", "(I)Z")
		// 		ok, err := jni.CallBooleanMethod(env, s, m, jni.Value(fd))
		// 		// TODO(bradfitz): return an error back up to netns if this fails, once
		// 		// we've had some experience with this and analyzed the logs over a wide
		// 		// range of Android phones. For now we're being paranoid and conservative
		// 		// and do the JNI call to protect best effort, only logging if it fails.
		// 		// The risk of returning an error is that it breaks users on some Android
		// 		// versions even when they're not using exit nodes. I'd rather the
		// 		// relatively few number of exit node users file bug reports if Tailscale
		// 		// doesn't work and then we can look for this log print.
		// 		if err != nil || !ok {
		// 			log.Printf("[unexpected] VpnService.protect(%d) = %v, %v", fd, ok, err)
		// 		}
		// 		return nil // even on error. see big TODO above.
		// 	})
		// })
		a.service = s
		return nil
	})
	return a.buildVPNConfigurationAndConnect()
}

func (a *androidApp) buildVPNConfigurationAndConnect() error {
	if string(a.nebulaConfig) == "" {
		return nil
	}
	config := nc.NewC(logrus.StandardLogger())
	if err := config.LoadString(string(a.nebulaConfig)); err != nil {
		return err
	}
	pki := config.GetMap("pki", nil)
	hostCert, _, err := cert.UnmarshalNebulaCertificateFromPEM([]byte(pki["cert"].(string)))
	if err != nil {
		panic(err)
	}
	return jni.Do(a.jvm, func(env *jni.Env) error {
		cls := jni.GetObjectClass(env, a.service)
		m := jni.GetMethodID(env, cls, "newBuilder", "()Landroid/net/VpnService$Builder;")
		b, err := jni.CallObjectMethod(env, a.service, m)
		if err != nil {
			return fmt.Errorf("PCloudVPNService.newBuilder: %v", err)
		}
		bcls := jni.GetObjectClass(env, b)
		addAddress := jni.GetMethodID(env, bcls, "addAddress", "(Ljava/lang/String;I)Landroid/net/VpnService$Builder;")
		addRoute := jni.GetMethodID(env, bcls, "addRoute", "(Ljava/lang/String;I)Landroid/net/VpnService$Builder;")
		for _, ipNet := range hostCert.Details.Ips {
			ip := ipNet.IP.String()
			prefix, _ := ipNet.Mask.Size()
			_, err := jni.CallObjectMethod(
				env,
				b,
				addAddress,
				jni.Value(jni.JavaString(env, ip)),
				jni.Value(jni.Value(prefix)))
			if err != nil {
				return err
			}
			_, err = jni.CallObjectMethod(
				env,
				b,
				addRoute,
				jni.Value(jni.JavaString(env, ip)),
				jni.Value(jni.Value(prefix)))
		}
		tun := config.GetMap("tun", nil)
		setMtu := jni.GetMethodID(env, bcls, "setMtu", "(I)Landroid/net/VpnService$Builder;")
		if _, err := jni.CallObjectMethod(env, b, setMtu, jni.Value(tun["mtu"].(int))); err != nil {
			return err
		}
		establish := jni.GetMethodID(env, bcls, "establish", "()Landroid/os/ParcelFileDescriptor;")
		parcelFD, err := jni.CallObjectMethod(env, b, establish)
		if err != nil {
			return err
		}
		parcelCls := jni.GetObjectClass(env, parcelFD)
		// detachFd := jni.GetMethodID(env, parcelCls, "detachFd", "()I")
		// tunFD, err := jni.CallIntMethod(env, parcelFD, detachFd)
		// if err != nil {
		// 	return fmt.Errorf("detachFd: %v", err)
		// }
		// fd := int(tunFD)
		// protect := jni.GetMethodID(env, cls, "protect", "(I)Z")
		// ok, err := jni.CallBooleanMethod(env, a.service, protect, jni.Value(fd))
		// if err != nil || !ok {
		// 	return fmt.Errorf("protect: %v %v", err, ok)
		// }
		getFd := jni.GetMethodID(env, parcelCls, "getFd", "()I")
		tunFD, err := jni.CallIntMethod(env, parcelFD, getFd)
		if err != nil {
			return err
		}
		fd := int(tunFD)
		ctrl, err := nebula.Main(config, false, "pcloud", logrus.StandardLogger(), &fd)
		if err != nil {
			return err
		}
		ctrl.Start()
		a.ctrl = ctrl
		return nil
	})

}
