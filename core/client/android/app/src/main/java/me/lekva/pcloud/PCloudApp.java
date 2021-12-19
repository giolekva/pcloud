package me.lekva.pcloud;

import android.app.Application;
import android.app.NotificationChannel;
import android.content.SharedPreferences;
import android.os.Build;

import androidx.core.app.NotificationManagerCompat;
import androidx.security.crypto.EncryptedSharedPreferences;
import androidx.security.crypto.MasterKey;

import org.gioui.Gio;

import java.io.IOException;
import java.security.GeneralSecurityException;

public class PCloudApp extends Application {
    static final String STATUS_CHANNEL_ID = "pcloud-status";
    static final int STATUS_NOTIFICATION_ID = 1;

    static final String NOTIFY_CHANNEL_ID = "pcloud-notify";
    static final int NOTIFY_NOTIFICATION_ID = 2;

    @Override
    public void onCreate() {
        super.onCreate();
        Gio.init(this);

        createNotificationChannel(NOTIFY_CHANNEL_ID, "Notifications", NotificationManagerCompat.IMPORTANCE_DEFAULT);
        createNotificationChannel(STATUS_CHANNEL_ID, "VPN Status", NotificationManagerCompat.IMPORTANCE_LOW);
    }

    private void createNotificationChannel(String id, String name, int importance) {
        if (Build.VERSION.SDK_INT < Build.VERSION_CODES.O) {
            return;
        }
        NotificationChannel channel = new NotificationChannel(id, name, importance);
        NotificationManagerCompat nm = NotificationManagerCompat.from(this);
        nm.createNotificationChannel(channel);
    }

    // encryptToPref a byte array of data using the Jetpack Security
    // library and writes it to a global encrypted preference store.
    public void encryptToPref(String prefKey, String plaintext) throws IOException, GeneralSecurityException {
        getEncryptedPrefs().edit().putString(prefKey, plaintext).commit();
    }

    // decryptFromPref decrypts a encrypted preference using the Jetpack Security
    // library and returns the plaintext.
    public String decryptFromPref(String prefKey) throws IOException, GeneralSecurityException {
        return getEncryptedPrefs().getString(prefKey, null);
    }

    private SharedPreferences getEncryptedPrefs() throws IOException, GeneralSecurityException {
        MasterKey key = new MasterKey.Builder(this)
                .setKeyScheme(MasterKey.KeyScheme.AES256_GCM)
                .build();
        return EncryptedSharedPreferences.create(
                this,
                "secret_shared_prefs",
                key,
                EncryptedSharedPreferences.PrefKeyEncryptionScheme.AES256_SIV,
                EncryptedSharedPreferences.PrefValueEncryptionScheme.AES256_GCM
        );
    }
}
