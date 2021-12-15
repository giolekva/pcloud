package me.lekva.pcloud;

import android.app.Service;
import android.content.Intent;
import android.net.VpnService;
import android.os.Handler;
import android.os.Message;

import androidx.annotation.NonNull;

public class PCloudVPNService extends VpnService implements Handler.Callback {
    public static final String ACTION_CONNECT = "CONNECT";
    public static final String ACTION_DISCONNECT = "DISCONNECT";

    private boolean running = false;
    private Handler handler = null;

    @Override
    public void onCreate() {
        if (handler == null) {
            handler = new Handler(this);
        }
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if (intent != null && intent.getAction().equals(ACTION_DISCONNECT)) {
            stopVpn();
            return Service.START_NOT_STICKY;
        } else {
            startVpn();
            return Service.START_STICKY;
        }
    }

    @Override
    public void onDestroy() {
        stopVpn();
    }

    private void startVpn() {
        System.out.println("--- START");
    }

    private void stopVpn() {
        System.out.println("--- STOP");
        running = false;
    }

    @Override
    public boolean handleMessage(@NonNull Message message) {
        System.out.println(getString(message.what));
        return true;
    }
}
