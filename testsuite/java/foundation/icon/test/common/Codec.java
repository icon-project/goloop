package foundation.icon.test.common;

import foundation.icon.ee.io.*;

public abstract class Codec {
    abstract public DataReader newReader(byte[] bytes);
    abstract public DataWriter newWriter();

    public static final Codec messagePack = new Codec() {
        @Override
        public DataReader newReader(byte[] bytes) {
            return new MessagePackDataReader(bytes);
        }
        @Override
        public DataWriter newWriter() {
            return new MessagePackDataWriter();
        }
    };

    public static final Codec rlp = new Codec() {
        @Override
        public DataReader newReader(byte[] bytes) {
            return new RLPNDataReader(bytes);
        }

        @Override
        public DataWriter newWriter() {
            return new RLPNDataWriter();
        }
    };
}
