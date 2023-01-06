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
import java.util.Arrays;

public class RLPDataWriter extends AbstractRLPDataWriter implements DataWriter {
    @Override
    protected byte[] toByteArray(BigInteger bi) {
        if (bi.signum() < 0) {
            throw new IllegalArgumentException("cannot encode negative BigInteger");
        }
        var ba = bi.toByteArray();
        if (ba[0] == 0) {
            return Arrays.copyOfRange(ba, 1, ba.length);
        }
        return ba;
    }

    @Override
    protected void writeNullity(ByteArrayBuilder os, boolean nullity) {
        throw new UnsupportedOperationException("Cannot write null or nullable in RLP codec");
    }
}
