/*
 * Copyright 2020 ICON Foundation
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

package foundation.icon.ee.types;

import java.math.BigInteger;
import java.util.Map;

public class StepCost {
    public static final String GET = "get";
    public static final String REPLACE = "replace";
    public static final String EVENT_LOG = "eventLog";
    public static final String DEFAULT_GET = "defaultGet";
    public static final String DEFAULT_SET = "defaultSet";
    public static final String REPLACE_BASE = "replaceBase";
    public static final String DEFAULT_DELETE = "defaultDelete";
    public static final String EVENT_LOG_BASE = "eventLogBase";

    private final Map<String, BigInteger> costMap;

    public StepCost(Map<String, BigInteger> costMap) {
        this.costMap = costMap;
    }

    public boolean has(String key) {
        return costMap.containsKey(key);
    }

    public int value(String key) {
        return costMap.getOrDefault(key, BigInteger.ZERO).intValue();
    }
    public int get() {
        return value(GET);
    }

    public int replace() {
        return value(REPLACE);
    }

    public int eventLog() {
        return value(EVENT_LOG);
    }

    public int defaultGet() {
        return value(DEFAULT_GET);
    }

    public int defaultSet() {
        return value(DEFAULT_SET);
    }

    public int replaceBase() {
        return value(REPLACE_BASE);
    }

    public int defaultDelete() {
        return value(DEFAULT_DELETE);
    }

    public int eventLogBase() {
        return value(EVENT_LOG_BASE);
    }
}
