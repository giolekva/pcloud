package me.lekva.pcloud;

import com.google.gson.annotations.JsonAdapter;

public class VerifyRequest {
    @JsonAdapter(Base64TypeAdapter.class)
    public byte[] message;
    @JsonAdapter(Base64TypeAdapter.class)
    public byte[] signature;
}
