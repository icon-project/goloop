package org.aion.avm.userlib;


/**
 * Just a wrapper over the way we serialize/deserialize the code+arguments tuple for a CREATE call.
 * Specifically, this is encoded as:  4(code length), n(code), [4(args length), n(args)]
 * Note that null args are encoded as code only.
 */
public class CodeAndArguments {
    /**
     * Decodes the CodeAndArguments structure from the given byte[].
     * 
     * @param bytes The bytes to parse.
     * @return The CodeAndArguments structure or null if bytes couldn't be parsed as such a structure.
     */
    public static CodeAndArguments decodeFromBytes(byte[] bytes) {
        CodeAndArguments result = null;
        if ((null != bytes) && (bytes.length > 4)) {
            AionBuffer buffer = AionBuffer.wrap(bytes);
            int codeLength = buffer.getInt();

            if (codeLength + 4 <= bytes.length) {
                byte[] code = new byte[codeLength];
                buffer.get(code);
                byte[] arguments = null;

                if (codeLength + 8 <= bytes.length) {
                    int argLength = buffer.getInt();
                    if (codeLength + 8 + argLength == bytes.length) {
                        arguments = new byte[argLength];
                        buffer.get(arguments);
                    }
                }
                result = new CodeAndArguments(code, arguments);
            }
        }
        return result;
    }


    public final byte[] code;
    public final byte[] arguments;

    /**
     * Creates a new CodeAndArguments for the given code and arguments.
     * 
     * @param code The code (must NOT be null)
     * @param arguments The arguments (CAN be null)
     */
    public CodeAndArguments(byte[] code, byte[] arguments) {
        // Null code is a usage error but the arguments can be null.
        if (null == code) {
            throw new NullPointerException();
        }

        this.code = code;
        this.arguments = arguments;
    }

    /**
     * Encodes the receiver as a byte[].
     * 
     * @return The byte[] which can later be deserialized with decodeFromBytes(), above.
     */
    public byte[] encodeToBytes() {
        // Allocate the appropriate buffer.
        int bufferLength = (null != this.arguments)
                ? (4 + this.code.length + 4 + this.arguments.length)
                : (4 + this.code.length);
        AionBuffer buffer = AionBuffer.allocate(bufferLength);

        // Write the code.
        buffer.putInt(this.code.length);
        buffer.put(this.code);

        // Write the arguments.
        if (null != this.arguments) {
            buffer.putInt(this.arguments.length);
            buffer.put(this.arguments);
        }

        // Return the raw bytes.
        return buffer.getArray();
    }
}
