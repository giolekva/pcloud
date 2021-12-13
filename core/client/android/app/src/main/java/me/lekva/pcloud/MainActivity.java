package me.lekva.pcloud;

import android.Manifest;
import android.app.Activity;
import android.content.pm.PackageManager;
import android.os.Build;
import android.os.Bundle;
import android.view.View;

import androidx.activity.result.ActivityResultLauncher;
import androidx.annotation.RequiresApi;
import androidx.appcompat.app.AppCompatActivity;
import androidx.core.app.ActivityCompat;
import androidx.core.content.ContextCompat;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import com.journeyapps.barcodescanner.ScanContract;
import com.journeyapps.barcodescanner.ScanOptions;

import java.io.BufferedOutputStream;
import java.io.IOException;
import java.io.OutputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.nio.charset.StandardCharsets;

public class MainActivity extends AppCompatActivity {
    private final ActivityResultLauncher<ScanOptions> barcodeLauncher = registerForActivityResult(new ScanContract(),
            result -> {
                if(result.getContents() != null) {
                    Gson gson = new GsonBuilder().disableHtmlEscaping().create();
                    VPNApiServerConfig config = gson.fromJson(result.getContents(), VPNApiServerConfig.class);
                    join(config);
                }
            });

    private void join(VPNApiServerConfig config) {
        new Thread(() -> {
            VerifyRequest req = new VerifyRequest();
            req.message = config.message;
            req.signature = config.signature;
            Gson gson = new GsonBuilder().disableHtmlEscaping().create();
            byte[] data = gson.toJson(req).getBytes(StandardCharsets.UTF_8);
            HttpURLConnection urlConnection = null;
            try {
                URL url = new URL(config.address + "/api/verify");
                urlConnection = (HttpURLConnection) url.openConnection();
                urlConnection.setRequestMethod("POST");
                urlConnection.setRequestProperty("Content-Type", "application/json; charset=utf-8");
                urlConnection.setRequestProperty("Connection", "close");

                urlConnection.setInstanceFollowRedirects(false);
                urlConnection.setDoOutput(true);
                urlConnection.setFixedLengthStreamingMode(data.length);

                OutputStream out = new BufferedOutputStream(urlConnection.getOutputStream());
                out.write(data);
                out.flush();
                out.close();
                System.out.println(urlConnection.getResponseCode());
            } catch (IOException e) {
                e.printStackTrace();
            } finally {
                urlConnection.disconnect();
            }
        }).start();
    }

    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        Activity act = this;
        findViewById(R.id.scan_qr_code).setOnClickListener(new View.OnClickListener() {
            @RequiresApi(api = Build.VERSION_CODES.M)
            @Override
            public void onClick(View view) {
                if (ContextCompat.checkSelfPermission(MainActivity.this, Manifest.permission.INTERNET) != PackageManager.PERMISSION_GRANTED) {
                    System.out.println("fooooooo");
                    ActivityCompat.requestPermissions(MainActivity.this, new String[]{Manifest.permission.INTERNET}, 20);
                }

                ScanOptions options = new ScanOptions();
                options.setDesiredBarcodeFormats(ScanOptions.QR_CODE);
                options.setPrompt("Join PCloud network");
                options.setCameraId(0);  // Use a specific camera of the device
                options.setBeepEnabled(false);
                options.setBarcodeImageEnabled(false);
                barcodeLauncher.launch(options);
            }
        });
    }
}