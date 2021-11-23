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

    public int value(String key) {
        return costMap.getOrDefault(key, BigInteger.ZERO).intValue();
    }

    public int get() {
        return value(GET);
    }

    public int set() {
        return value(SET);
    }

    public int delete() {
        return value(DELETE);
    }

    public int log() {
        return value(LOG);
    }

    public int getBase() {
        return value(GET_BASE);
    }

    public int setBase() {
        return value(SET_BASE);
    }

    public int deleteBase() {
        return value(DELETE_BASE);
    }

    public int logBase() {
        return value(LOG_BASE);
    }

    public int replaceBase() {
        return (setBase() + deleteBase()) / 2;
    }

    public int getStorage(int prevLen) {
        return getBase() + prevLen * get();
    }

    public int setStorageSet(int newLen) {
        return setBase() + newLen * set();
    }

    public int setStorageReplace(int prevLen, int newLen) {
        return replaceBase() + prevLen * delete() + newLen * set();
    }

    public int setStorageDelete(int prevLen) {
        return deleteBase() + prevLen * delete();
    }

    public int eventLog(int len) {
        return logBase() + len * log();
    }
}
