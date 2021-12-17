package me.lekva.pcloud;

import android.content.Intent;
import android.content.res.Configuration;
import android.net.VpnService;
import android.os.Bundle;

import androidx.activity.result.ActivityResultLauncher;
import androidx.annotation.Nullable;
import androidx.appcompat.app.AppCompatActivity;

import com.journeyapps.barcodescanner.ScanContract;
import com.journeyapps.barcodescanner.ScanOptions;

import org.gioui.GioView;

public class PCloudActivity extends AppCompatActivity {
    private static final int VPN_START_CODE = 0x10;

    private GioView view;

    private final ActivityResultLauncher<ScanOptions> barcodeLauncher = registerForActivityResult(new ScanContract(),
            result -> {
                if(result.getContents() != null) {
                    qrcodeScanned(result.getContents());
                }
            });

    @Override
    public void onCreate(@Nullable Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        view = new GioView(this);
        setContentView(view);
    }

    @Override
    public void onDestroy() {
        view.destroy();
        super.onDestroy();
    }

    @Override
    public void onStart() {
        super.onStart();
        view.start();
    }

    @Override
    public void onStop() {
        view.stop();
        super.onStop();
    }

    @Override
    public void onConfigurationChanged(Configuration c) {
        super.onConfigurationChanged(c);
        view.configurationChanged();
    }

    @Override
    public void onLowMemory() {
        super.onLowMemory();
        view.onLowMemory();
    }

    @Override
    public void onBackPressed() {
        if (!view.backPressed())
            super.onBackPressed();
    }

    // TODO(giolekva): return void instead of String
    public String launchBarcodeScanner() {
        ScanOptions options = new ScanOptions();
        options.setDesiredBarcodeFormats(ScanOptions.QR_CODE);
        options.setPrompt("Join PCloud mesh");
        options.setCameraId(0);  // Use a specific camera of the device
        options.setBeepEnabled(true);
        options.setBarcodeImageEnabled(false);
        barcodeLauncher.launch(options);
        return null;
    }

    public String startVpn(String ipCidr) {
        Intent intent = VpnService.prepare(this);
        if (intent != null) {
            System.out.println("#### STARTVPN");
            intent.setAction(PCloudVPNService.ACTION_CONNECT);
            startActivityForResult(intent, VPN_START_CODE);
        } else {
            intent = new Intent(this, PCloudVPNService.class);
            startService(intent);
        }

        return null;
    }

    private native void qrcodeScanned(String contents);
}
