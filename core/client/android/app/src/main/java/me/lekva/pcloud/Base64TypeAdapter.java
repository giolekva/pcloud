package me.lekva.pcloud;

import android.os.Build;
import android.util.Base64;

import androidx.annotation.RequiresApi;

import com.google.gson.TypeAdapter;
import com.google.gson.stream.JsonReader;
import com.google.gson.stream.JsonWriter;

import java.io.IOException;

public class Base64TypeAdapter extends TypeAdapter<byte[]> {
    @RequiresApi(api = Build.VERSION_CODES.O)
    @Override
    public void write(JsonWriter out, byte[] value) throws IOException {
        String val = Base64.encodeToString(value, Base64.NO_WRAP);
        // out.value(val.substring(0, val.length() - 1));
        out.value(val);
    }

    @RequiresApi(api = Build.VERSION_CODES.O)
    @Override
    public byte[] read(JsonReader in) throws IOException {
        return Base64.decode(in.nextString(), Base64.NO_WRAP);
    }
}