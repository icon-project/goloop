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
    public static final String SET = "set";
    public static final String DELETE = "delete";
    public static final String LOG = "log";
    public static final String GET_BASE = "getBase";
    public static final String SET_BASE = "setBase";
    public static final String DELETE_BASE = "deleteBase";
    public static final String LOG_BASE = "logBase";

    private final Map<String, BigInteger> costMap;

    public StepCost(Map<String, BigInteger> costMap) {
        this.costMap = costMap;
    }

    public boolean has(String key) {
        return costMap.containsKey(key);
    }

    public long value(String key) {
        return costMap.getOrDefault(key, BigInteger.ZERO).longValue();
    }

    public long get() {
        return value(GET);
    }

    public long set() {
        return value(SET);
    }

    public long delete() {
        return value(DELETE);
    }

    public long log() {
        return value(LOG);
    }

    public long getBase() {
        return value(GET_BASE);
    }

    public long setBase() {
        return value(SET_BASE);
    }

    public long deleteBase() {
        return value(DELETE_BASE);
    }

    public long logBase() {
        return value(LOG_BASE);
    }

    public long replaceBase() {
        return (setBase() + deleteBase()) / 2;
    }

    public long getStorage(int prevLen) {
        return getBase() + prevLen * get();
    }

    public long setStorageSet(int newLen) {
        return setBase() + newLen * set();
    }

    public long setStorageReplace(int prevLen, int newLen) {
        return replaceBase() + prevLen * delete() + newLen * set();
    }

    public long setStorageDelete(int prevLen) {
        return deleteBase() + prevLen * delete();
    }

    public long eventLog(int len) {
        return logBase() + len * log();
    }
}
