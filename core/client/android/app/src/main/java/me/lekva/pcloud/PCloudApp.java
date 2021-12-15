package me.lekva.pcloud;

import android.app.Application;

import org.gioui.Gio;

public class PCloudApp extends Application {
    @Override
    public void onCreate() {
        super.onCreate();
        Gio.init(this);
    }
}
