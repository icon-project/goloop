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
    public static final String TAG_GOVERNANCE = "governance";
    public static final String TAG_NORMAL = "normal";

    public static final BigInteger STATUS_SUCCESS = BigInteger.ONE;
    public static final BigInteger STATUS_FAIL = BigInteger.ZERO;

    public static final String CONTENT_TYPE_ZIP = "application/zip";
    public static final String CONTENT_TYPE_JAVA = "application/java";
    public static final String SCORE_ROOT = "./data/scores/";
    public static final String JAVA_SCORE_ROOT = "./data/scores/java/";

    public static final long DEFAULT_STEP_LIMIT = 9000000;
    public static final long DEFAULT_WAITING_TIME = 7000; // millisecond
    public static final BigInteger DEFAULT_BALANCE = new BigInteger("100000000");

    public static final Address CHAINSCORE_ADDRESS
            = new Address("cx0000000000000000000000000000000000000000");
    public static final Address GOV_ADDRESS
            = new Address("cx0000000000000000000000000000000000000001");
    public static final Address TREASURY_ADDRESS
            = new Address("cx1000000000000000000000000000000000000000");

    public static final String SCORE_STATUS_PENDING = "pending";
    public static final String SCORE_STATUS_ACTIVE = "active";
    public static final String SCORE_STATUS_REJECT = "rejected";

    public static final String SCORE_MULTISIG_PATH = SCORE_ROOT + "multisig_wallet";
    public static final String SCORE_STEPCOUNTER_PATH = SCORE_ROOT + "step_counter";
    public static final String SCORE_DB_STEP_PATH = SCORE_ROOT + "db_step";
    public static final String SCORE_CROWDSALE_PATH = SCORE_ROOT + "crowdsale";
    public static final String SCORE_SAMPLETOKEN_PATH = SCORE_ROOT + "sample_token";
    public static final String SCORE_HELLOWORLD_PATH = SCORE_ROOT + "hello_world";
    public static final String SCORE_HELLOWORLD_UPDATE_PATH = SCORE_ROOT + "hello_world2";
    public static final String SCORE_CHECKPARAMS_PATH = SCORE_ROOT + "check_params";
    public static final String SCORE_RECEIPT_PATH = SCORE_ROOT + "receipt";
    public static final String SCORE_API_PATH = SCORE_ROOT + "score_api";
    public static final String SCORE_GOV_PATH = "./data/genesisStorage/" + "governance";
    public static final String SCORE_GOV_UPDATE_PATH = SCORE_ROOT + "governance";
    public static final String JSCORE_MYSAMPLETOKEN = JAVA_SCORE_ROOT + "sampleToken.jar";
    public static final String JSCORE_APITEST = JAVA_SCORE_ROOT + "apiTest.jar";
}
