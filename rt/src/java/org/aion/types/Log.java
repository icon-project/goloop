package org.aion.types;

import java.util.ArrayList;
import java.util.Arrays;
import java.util.Collections;
import java.util.List;

/**
 * A log holds an address and some data, and optionally may associate topics with this data.
 *
 * A log is completely general and holds all of its information in terms of byte arrays.
 *
 * A valid log may have zero or more topics and must always have a non-null address and non-null data.
 *
 * A log is immutable.
 */
public final class Log {
    private final byte[] address;
    private final byte[] data;
    private final List<byte[]> topics;

    private Log(byte[] address, byte[] data, List<byte[]> topics) {
        if (address == null) {
            throw new NullPointerException("Cannot create log with null address!");
        }
        if (topics == null) {
            throw new NullPointerException("Cannot create log with null topics!");
        }
        if (data == null) {
            throw new NullPointerException("Cannot create log with null data!");
        }

        this.address = copyOf(address);
        this.topics = copyOfBytesList(topics);
        this.data = copyOf(data);
    }

    /**
     * Constructs a new log with only data and no topics.
     *
     * @param address The address that produced this log.
     * @param data The log data.
     * @return the log.
     */
    public static Log dataOnly(byte[] address, byte[] data) {
        return new Log(address, data, Collections.emptyList());
    }

    /**
     * Constructs a new log with data and the specified topics.
     *
     * @param address The address that produced this log.
     * @param topics The associated topics.
     * @param data The log data.
     * @return the log.
     */
    public static Log topicsAndData(byte[] address, List<byte[]> topics, byte[] data) {
        return new Log(address, data, topics);
    }

    /**
     * Returns a copy of the address that produced this log.
     *
     * @return the address that produced this log.
     */
    public byte[] copyOfAddress() {
        return copyOf(this.address);
    }

    /**
     * Returns a copy of the associated log topics.
     *
     * @return the associated log topics.
     */
    public List<byte[]> copyOfTopics() {
        return copyOfBytesList(this.topics);
    }

    /**
     * Returns a copy of the data.
     *
     * @return the data.
     */
    public byte[] copyOfData() {
        return copyOf(this.data);
    }

    /**
     * Returns {@code true} only if other is a {@link Log}, and the two logs are equal. Two logs
     * are equal only if they contain the same address, same data, and the same list of topics -
     * where the ordering of the topics does matter!
     *
     * Returns {@code false} otherwise.
     *
     * @param other The object whose equality with this is to be determined.
     * @return whether other is equal to this.
     */
    @Override
    public boolean equals(Object other) {
        if (!(other instanceof Log)) {
            return false;
        }
        if (other == this) {
            return true;
        }

        Log otherLog = (Log) other;
        if (!Arrays.equals(this.address, otherLog.address)) {
            return false;
        }
        if (!Arrays.equals(this.data, otherLog.data)) {
            return false;
        }
        if (this.topics.size() != otherLog.topics.size()) {
            return false;
        }
        for (int i = 0; i < this.topics.size(); i++) {
            if (!Arrays.equals(this.topics.get(i), otherLog.topics.get(i))) {
                return false;
            }
        }
        return true;
    }

    @Override
    public int hashCode() {
        int hash = Arrays.hashCode(this.address) + Arrays.hashCode(this.data);
        for (byte[] topic : this.topics) {
            hash += Arrays.hashCode(topic);
        }
        return hash;
    }

    @Override
    public String toString() {
        StringBuilder representation = new StringBuilder("Log { address = ")
            .append(this.address)
            .append(", data = ")
            .append(this.data)
            .append(", topics = [ ");
        for (int i = 0; i < this.topics.size(); i++) {
            representation.append(this.topics.get(i));
            if (i < this.topics.size() - 1) {
                representation.append(", ");
            }
        }
        return representation.append(" ]}").toString();
    }

    private static List<byte[]> copyOfBytesList(List<byte[]> bytesList) {
        List<byte[]> copy = new ArrayList<>();
        for (byte[] bytes : bytesList) {
            copy.add(copyOf(bytes));
        }
        return copy;
    }

    private static byte[] copyOf(byte[] bytes) {
        return Arrays.copyOf(bytes, bytes.length);
    }
}
