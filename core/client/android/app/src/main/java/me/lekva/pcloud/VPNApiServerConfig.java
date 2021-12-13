package me.lekva.pcloud;

import com.google.gson.annotations.JsonAdapter;
import com.google.gson.annotations.SerializedName;

public class VPNApiServerConfig {
    @SerializedName("vpn_api_addr")
    public String address;
    @JsonAdapter(Base64TypeAdapter.class)
    public byte[] message;
    @JsonAdapter(Base64TypeAdapter.class)
    public byte[] signature;
}
