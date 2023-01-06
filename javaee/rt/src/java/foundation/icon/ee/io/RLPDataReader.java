/*
 * Copyright 2023 ICON Foundation
 *
 * Licensed under the Apache License, Version 2.0 (the "License");
 * you may not use this file except in compliance with the License.
 * You may obtain a copy of the License at
 *
 *     http://www.apache.org/licenses/LICENSE-2.0
 *
 * Unless required by applicable law or agreed to in writing, software
 * distributed under the License is distributed on an "AS IS" BASIS,
 * WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
 * See the License for the specific language governing permissions and
 * limitations under the License.
 */

package foundation.icon.ee.io;

import java.math.BigInteger;

public class RLPDataReader extends AbstractRLPDataReader implements DataReader {
    public RLPDataReader(byte[] data) {
        super(data);
    }

    @Override
    protected int peekNull(byte[] ba, int offset, int len, boolean forRead) {
        if (forRead) {
            throw new UnsupportedOperationException("Cannot read null or nullable in RLP codec");
        }
        return 0;
    }

    @Override
    protected BigInteger peekBigInteger(byte[] ba, int offset, int len) {
        if (len == 0) {
            return BigInteger.ZERO;
        }
        return new BigInteger(1, ba, offset, len);
    }
}
