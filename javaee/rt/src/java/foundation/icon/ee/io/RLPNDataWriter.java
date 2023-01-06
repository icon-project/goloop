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

public class RLPNDataWriter extends AbstractRLPDataWriter implements DataWriter {

    @Override
    protected byte[] toByteArray(BigInteger bi) {
        return bi.toByteArray();
    }

    @Override
    protected void writeNullity(ByteArrayBuilder os, boolean nullity) {
        if (nullity) {
            os.write(0xf8);
            os.write(0x00);
        }
    }
}
