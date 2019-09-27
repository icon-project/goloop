package org.aion.avm.core.util;

import java.util.Arrays;

/**
 * A wrapper of byte[], to be used as the HashMap key.
 */
public final class ByteArrayWrapper
{
    private final byte[] data;

    public ByteArrayWrapper(byte[] data)
    {
        if (data == null)
        {
            throw new NullPointerException();
        }
        this.data = data;
    }

    @Override
    public boolean equals(Object object)
    {
        if (!(object instanceof ByteArrayWrapper))
        {
            return false;
        }
        return Arrays.equals(data, ((ByteArrayWrapper)object).data);
    }

    @Override
    public int hashCode()
    {
        return Arrays.hashCode(data);
    }

    @Override
    public String toString() {
        return Helpers.bytesToHexString(data);
    }
}
