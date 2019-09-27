package org.aion.avm.core.util;


/**
 * Used to resize the log topics as per the specification defined in issue-358:
 * -null is invalid and results in NullPointerException
 * -any byte[] longer then 32 bytes is truncated to 32 by removing bytes at the end (higher
 *  indices removed while indices 0 to 31 remain untouched)
 * -any byte[] shorter than 32 bytes is zero-extended by adding bytes at the end (for byte[]
 *  of length n given, adding 0 bytes from index n to 31 while leaving indices 0 to n-1 untouched)
 */
public class LogSizeUtils {
    public static int TOPIC_SIZE = 32;

    /**
     * Truncates or pads the given input topic to 32 bytes and returns a new copy.
     * 
     * @param topic The log topic to truncate or pad.
     * @return The 32-byte interpretation of the input topic.
     */
    public static byte[] truncatePadTopic(byte[] topic) {
        byte[] result = new byte[TOPIC_SIZE];
        if (null == topic) {
            throw new NullPointerException();
        } else if (topic.length < TOPIC_SIZE) {
            // Too short:  zero-pad.
            System.arraycopy(topic, 0, result, 0, topic.length);
            for (int i = topic.length; i < TOPIC_SIZE; ++i) {
                result[i] = 0;
            }
        } else if (topic.length > TOPIC_SIZE) {
            // Too long:  truncate.
            System.arraycopy(topic, 0, result, 0, TOPIC_SIZE);
        } else {
            // Just the right size.
            System.arraycopy(topic, 0, result, 0, TOPIC_SIZE);
        }
        return result;
    }
}
