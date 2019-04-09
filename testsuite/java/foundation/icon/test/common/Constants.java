/*
 * Copyright (c) 2019 ICON Foundation
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

package foundation.icon.test.common;

import foundation.icon.icx.data.Address;

import java.math.BigInteger;

public class Constants {
    public static final BigInteger STATUS_SUCCESS = BigInteger.ONE;
    public static final Address ZERO_ADDRESS = new Address("cx0000000000000000000000000000000000000000");
    public static final String CONTENT_TYPE = "application/zip";
    public static final String SCORE_ROOT = "./data/";
    public static final long DEFAULT_WAITING_TIME = 7000; // millisecond
    public static final Address CHAINSCORE_ADDRESS
            = new Address("cx0000000000000000000000000000000000000000");
    public static final Address GOV_ADDRESS
            = new Address("cx0000000000000000000000000000000000000001");
}
