/*
 * Copyright 2019 ICON Foundation
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
    public static final String TAG_PY_SCORE = "pyScore";
    public static final String TAG_PY_GOV = "pyGov";
    public static final String TAG_JAVA_SCORE = "javaScore";
    public static final String TAG_JAVA_GOV = "javaGov";

    public static final BigInteger STATUS_SUCCESS = BigInteger.ONE;
    public static final BigInteger STATUS_FAILURE = BigInteger.ZERO;

    public static final BigInteger DEFAULT_STEPS = BigInteger.valueOf(200000);
    public static final long DEFAULT_WAITING_TIME = 7000;

    public static final String CONTENT_TYPE_PYTHON = "application/zip";
    public static final String CONTENT_TYPE_JAVA = "application/java";

    public static final Address CHAINSCORE_ADDRESS
            = new Address("cx0000000000000000000000000000000000000000");
    public static final Address GOV_ADDRESS
            = new Address("cx0000000000000000000000000000000000000001");
    public static final Address TREASURY_ADDRESS
            = new Address("hx1000000000000000000000000000000000000000");
}
