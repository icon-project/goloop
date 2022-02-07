/*
 * Copyright 2022 ICON Foundation
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

package testcases;

import score.Address;
import score.ArrayDB;
import score.Context;
import score.DictDB;
import score.VarDB;
import score.annotation.External;

import java.math.BigInteger;
import java.util.List;
import java.util.Map;

public class ContainerDB {
    private final VarDB<BigInteger> varInt = Context.newVarDB("var_int", BigInteger.class);
    private final VarDB<String> varStr = Context.newVarDB("var_str", String.class);
    private final VarDB<byte[]> varBytes = Context.newVarDB("var_bytes", byte[].class);
    private final VarDB<Boolean> varBool = Context.newVarDB("var_bool", Boolean.class);
    private final VarDB<Address> varAddr = Context.newVarDB("var_addr", Address.class);

    private final DictDB<String, BigInteger> dictInt = Context.newDictDB("dict_int", BigInteger.class);
    private final DictDB<String, String> dictStr = Context.newDictDB("dict_str", String.class);
    private final DictDB<String, byte[]> dictBytes = Context.newDictDB("dict_bytes", byte[].class);
    private final DictDB<String, Boolean> dictBool = Context.newDictDB("dict_bool", Boolean.class);
    private final DictDB<String, Address> dictAddr = Context.newDictDB("dict_addr", Address.class);

    private final ArrayDB<BigInteger> arrInt = Context.newArrayDB("arr_int", BigInteger.class);
    private final ArrayDB<String> arrStr = Context.newArrayDB("arr_str", String.class);
    private final ArrayDB<byte[]> arrBytes = Context.newArrayDB("arr_bytes", byte[].class);
    private final ArrayDB<Boolean> arrBool = Context.newArrayDB("arr_bool", Boolean.class);
    private final ArrayDB<Address> arrAddr = Context.newArrayDB("arr_addr", Address.class);

    @External(readonly=true)
    public Map<String, Object> getVar(String type) {
        switch (type) {
            case "int":
                return Map.of(type, varInt.get());
            case "str":
                return Map.of(type, varStr.get());
            case "bytes":
                return Map.of(type, varBytes.get());
            case "bool":
                return Map.of(type, varBool.get());
            case "addr":
                return Map.of(type, varAddr.get());
        }
        return Map.of();
    }

    @External(readonly=true)
    public Map<String, Object> getDict(String key, String type) {
        try {
            switch (type) {
                case "int":
                    return Map.of(key, dictInt.get(key));
                case "str":
                    return Map.of(key, dictStr.get(key));
                case "bytes":
                    return Map.of(key, dictBytes.get(key));
                case "bool":
                    return Map.of(key, dictBool.get(key));
                case "addr":
                    return Map.of(key, dictAddr.get(key));
            }
        } catch (NullPointerException e) {
            // just catch NPE when there is no corresponding key in the DictDB
        }
        return Map.of();
    }

    @External(readonly=true)
    public List<Object> getArray(String type) {
        switch (type) {
            case "int": {
                int size = arrInt.size();
                BigInteger[] array = new BigInteger[size];
                for (int i = 0; i < size; i++) {
                    array[i] = arrInt.get(i);
                }
                return List.of((Object[]) array);
            }
            case "str": {
                int size = arrStr.size();
                String[] array = new String[size];
                for (int i = 0; i < size; i++) {
                    var item = arrStr.get(i);
                    array[i] = item == null ? "" : item;
                }
                return List.of((Object[]) array);
            }
            case "bytes": {
                int size = arrBytes.size();
                byte[][] array = new byte[size][];
                for (int i = 0; i < size; i++) {
                    array[i] = arrBytes.get(i);
                }
                return List.of((Object[]) array);
            }
            case "bool": {
                int size = arrBool.size();
                Boolean[] array = new Boolean[size];
                for (int i = 0; i < size; i++) {
                    array[i] = arrBool.get(i);
                }
                return List.of((Object[]) array);
            }
            case "addr": {
                int size = arrAddr.size();
                Address[] array = new Address[size];
                for (int i = 0; i < size; i++) {
                    array[i] = arrAddr.get(i);
                }
                return List.of((Object[]) array);
            }
        }
        return List.of();
    }
}
