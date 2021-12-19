package me.lekva.pcloud;

import android.app.PendingIntent;
import android.app.Service;
import android.content.Intent;
import android.net.VpnService;
import android.os.Build;
import android.os.Handler;
import android.os.Message;
import android.system.OsConstants;

import androidx.annotation.NonNull;
import androidx.annotation.RequiresApi;
import androidx.core.app.NotificationCompat;

public class PCloudVPNService extends VpnService implements Handler.Callback {
    public static final String ACTION_CONNECT = "CONNECT";
    public static final String ACTION_DISCONNECT = "DISCONNECT";

    private PendingIntent configureIntent;

    private boolean running = false;
    private Handler handler = null;

    @Override
    public void onCreate() {
        if (handler == null) {
            handler = new Handler(this);
        }

        configureIntent = PendingIntent.getActivity(this, 0, new Intent(this, PCloudActivity.class),
                PendingIntent.FLAG_UPDATE_CURRENT);
    }

    @Override
    public int onStartCommand(Intent intent, int flags, int startId) {
        if (intent != null && ACTION_DISCONNECT.equals(intent.getAction())) {
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

    @RequiresApi(api = Build.VERSION_CODES.O)
    private void startVpn() {
        System.out.println("--- START");
        updateForegroundNotification();
        connect();
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

    private PendingIntent configIntent() {
        return PendingIntent.getActivity(this, 0, new Intent(this, PCloudActivity.class), PendingIntent.FLAG_UPDATE_CURRENT);
    }

    public VpnService.Builder newBuilder() {
        VpnService.Builder builder = new VpnService.Builder()
                .setConfigureIntent(configIntent())
                .allowFamily(OsConstants.AF_INET)
                .allowFamily(OsConstants.AF_INET6);
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.Q)
            builder.setMetered(false); // Inherit the metered status from the underlying networks.
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.M)
            builder.setUnderlyingNetworks(null); // Use all available networks.
        return builder;
    }

    private void updateForegroundNotification() {
        NotificationCompat.Builder builder = new NotificationCompat.Builder(this, PCloudApp.STATUS_CHANNEL_ID)
                .setContentTitle("PCloud")
                .setContentText("hiiii")
                .setContentIntent(configureIntent)
                .setPriority(NotificationCompat.PRIORITY_LOW);
        startForeground(PCloudApp.STATUS_NOTIFICATION_ID, builder.build());
    }

    private native void connect();
    private native void disconnect();
}
