package foundation.icon.icx.data;

public class Base64 {
    String data;
    public Base64(String data) {
        this.data = data;
    }

    public byte[] decode() {
        return java.util.Base64.getDecoder().decode(data);
    }

}
