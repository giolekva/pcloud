run:
	go run github.com/giolekva/pcloud/core/client/cmd/pcloud

aar: export ANDROID_SDK_ROOT=/Users/lekva/Library/Android/sdk/
aar:
	go run gioui.org/cmd/gogio -target android -buildmode archive -o android/app/libs/pcloud.aar github.com/giolekva/pcloud/core/client/cmd/pcloud
