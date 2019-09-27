package org.aion.avm.core.util;

import i.RuntimeAssertionError;


/**
 * A simple state machine for parsing descriptors.  This can be used to parse both simple type descriptors and complete method descriptors.
 * Note that the design takes and returns a userData argument in order to avoid the need to create stateful Callbacks implementation classes
 * in the common cases.  The userData returned from one event is passed as the input to the next (with the first coming from outside and the
 * last being returned).
 */
public class DescriptorParser {
    public static final char BYTE = 'B';
    public static final char CHAR = 'C';
    public static final char DOUBLE = 'D';
    public static final char FLOAT = 'F';
    public static final char INTEGER = 'I';
    public static final char LONG = 'J';
    public static final char SHORT = 'S';
    public static final char BOOLEAN = 'Z';
    public static final char ARRAY = '[';
    public static final char OBJECT_START = 'L';
    public static final char OBJECT_END = ';';

    public static final char ARGS_START = '(';
    public static final char ARGS_END = ')';
    public static final char VOID = 'V';


    /**
     * Parses the entire descriptor, calling out to the given callbacks object to handle parsing events.
     * 
     * @param descriptor The descriptor to parse.
     * @param callbacks The Callbacks object which will handle the parsing events.
     * @param userData The initial userData object to pass to the Callbacks methods.
     * @return The final userData returned by the final Callbacks event.
     */
    public static <T> T parse(String descriptor, Callbacks<T> callbacks, T userData) {
        int arrayDimensions = 0;
        StringBuilder parsingObject = null;
        
        for (int index = 0; index < descriptor.length(); ++index) {
            char c = descriptor.charAt(index);
            if (null != parsingObject) {
                switch (c) {
                case OBJECT_END:
                    // End of object name.
                    userData = callbacks.readObject(arrayDimensions, parsingObject.toString(), userData);
                    arrayDimensions = 0;
                    parsingObject = null;
                    break;
                default:
                    // Just assemble the object.
                    parsingObject.append(c);
                }
            } else {
                switch (c) {
                case BYTE:
                    userData = callbacks.readByte(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case CHAR:
                    userData = callbacks.readChar(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case DOUBLE:
                    userData = callbacks.readDouble(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case FLOAT:
                    userData = callbacks.readFloat(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case INTEGER:
                    userData = callbacks.readInteger(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case LONG:
                    userData = callbacks.readLong(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case SHORT:
                    userData = callbacks.readShort(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case BOOLEAN:
                    userData = callbacks.readBoolean(arrayDimensions, userData);
                    arrayDimensions = 0;
                    break;
                case VOID:
                    RuntimeAssertionError.assertTrue(0 == arrayDimensions);
                    userData = callbacks.readVoid(userData);
                    break;
                case OBJECT_START:
                    parsingObject = new StringBuilder();
                    break;
                case ARRAY:
                    arrayDimensions += 1;
                    break;
                case ARGS_START:
                    RuntimeAssertionError.assertTrue(0 == arrayDimensions);
                    userData = callbacks.argumentStart(userData);
                    break;
                case ARGS_END:
                    RuntimeAssertionError.assertTrue(0 == arrayDimensions);
                    userData = callbacks.argumentEnd(userData);
                    break;
                default:
                    throw RuntimeAssertionError.unreachable("Unexpected descriptor character: " + c);
                }
            }
        }
        RuntimeAssertionError.assertTrue(0 == arrayDimensions);
        RuntimeAssertionError.assertTrue(null == parsingObject);
        return userData;
    }


    /**
     * The callback interface which defines the parsing events generated by this parser.
     * 
     * @param <T> The userData arg/return type.
     */
    public static interface Callbacks<T> {
        public T argumentStart(T userData);
        public T argumentEnd(T userData);

        public T readObject(int arrayDimensions, String type, T userData);

        public T readVoid(T userData);

        public T readBoolean(int arrayDimensions, T userData);

        public T readShort(int arrayDimensions, T userData);

        public T readLong(int arrayDimensions, T userData);

        public T readInteger(int arrayDimensions, T userData);

        public T readFloat(int arrayDimensions, T userData);

        public T readDouble(int arrayDimensions, T userData);

        public T readChar(int arrayDimensions, T userData);

        public T readByte(int arrayDimensions, T userData);
    }


    /**
     * A specialized implementation of Callbacks which omits events related to argument lists.
     * This is a more appropriate implementation to use if the caller is only parsing single types.
     * 
     * @param <T> The userData arg/return type.
     */
    public static abstract class TypeOnlyCallbacks<T> implements Callbacks<T> {
        public T argumentStart(T userData) {
            throw RuntimeAssertionError.unreachable("Type-only parser received method-style callback");
        }
        public T argumentEnd(T userData) {
            throw RuntimeAssertionError.unreachable("Type-only parser received method-style callback");
        }

        public T readVoid(T userData) {
            throw RuntimeAssertionError.unreachable("Type-only parser received method-style callback");
        }
    }
}
