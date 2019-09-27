package i;

import java.nio.charset.Charset;
import java.nio.charset.StandardCharsets;


/**
 * Many of our classes are serialized using similar mechanisms so this class exists to contain those implementations and avoid duplication.
 */
public final class CodecIdioms {
    private static final Charset SERIALIZATION_CHARSET = StandardCharsets.UTF_8;

    public static String deserializeString(IObjectDeserializer deserializer) {
        int length = deserializer.readInt();
        byte[] data = new byte[length];
        deserializer.readByteArray(data);
        return new String(data, SERIALIZATION_CHARSET);
    }

    public static void serializeString(IObjectSerializer serializer, String string) {
        byte[] data = string.getBytes(SERIALIZATION_CHARSET);
        serializer.writeInt(data.length);
        serializer.writeByteArray(data);
    }

    public static byte[] deserializeByteArray(IObjectDeserializer deserializer) {
        int length = deserializer.readInt();
        byte[] array = new byte[length];
        deserializer.readByteArray(array);
        return array;
    }

    public static void serializeByteArray(IObjectSerializer serializer, byte[] array) {
        serializer.writeInt(array.length);
        serializer.writeByteArray(array);
    }
    
    public static boolean[] deserializeBooleanArray(IObjectDeserializer deserializer) {
        int length = deserializer.readInt();
        boolean[] array = new boolean[length];
        for (int i = 0; i < length; ++i) {
            array[i] = deserializer.readBoolean();
        }
        return array;
    }

    public static void serializeBooleanArray(IObjectSerializer serializer, boolean[] array) {
        serializer.writeInt(array.length);
        for (int i = 0; i < array.length; ++i) {
            serializer.writeBoolean(array[i]);
        }
    }
}
