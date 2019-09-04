package org.aion.avm.userlib;

/**
 * A collection of methods to facilitate contract development.
 */
public class AionUtilities {

    /**
     * Returns a new byte array of length 32 that right-aligns the input bytes by padding them on the left with 0.
     * Note that the input is not truncated if it is larger than 32 bytes.
     * This method can be used to pad log topics.
     *
     * @param topic bytes to pad
     * @return Zero padded topic
     * @throws NullPointerException if topic is null
     */
    public static byte[] padLeft(byte[] topic) {
        int topicSize = 32;
        byte[] result;
        if (null == topic) {
            throw new NullPointerException();
        } else if (topic.length < topicSize) {
            result = new byte[topicSize];
            System.arraycopy(topic, 0, result, topicSize - topic.length, topic.length);
        } else {
            // if topic is larger than 32 bytes or the right size
            result = topic;
        }
        return result;
    }
}
