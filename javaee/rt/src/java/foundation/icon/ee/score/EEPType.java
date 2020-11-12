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

package foundation.icon.ee.score;

import foundation.icon.ee.types.Method;
import foundation.icon.ee.util.Containers;
import org.objectweb.asm.Type;
import score.Address;

import java.math.BigInteger;
import java.util.List;
import java.util.Map;

public class EEPType {
    @SuppressWarnings("unchecked")
    private static final Map.Entry<Type, Integer>[] commonMap = new Map.Entry[]{
            Map.entry(Type.getType(byte.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(char.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(short.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(int.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(long.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(BigInteger.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(String.class), Method.DataType.STRING),
            Map.entry(Type.getType(byte[].class), Method.DataType.BYTES),
            Map.entry(Type.getType(boolean.class), Method.DataType.BOOL),
            Map.entry(Type.getType(Address.class), Method.DataType.ADDRESS),
    };

    @SuppressWarnings("unchecked")
    private static final Map.Entry<Type, Integer>[] returnMap = new Map.Entry[]{
            Map.entry(Type.getType(void.class), Method.DataType.NONE),
            Map.entry(Type.getType(List.class), Method.DataType.LIST),
            Map.entry(Type.getType(Map.class), Method.DataType.DICT),
    };

    @SuppressWarnings("unchecked")
    private static final Map.Entry<Type, Integer>[] fieldMap = new Map.Entry[]{
            Map.entry(Type.getType(Byte.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(Character.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(Short.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(Integer.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(Long.class), Method.DataType.INTEGER),
            Map.entry(Type.getType(Boolean.class), Method.DataType.BOOL),
    };

    public static final Map<Type, Integer> fromBasicParamType = Map.ofEntries(
            commonMap
    );

    public static final Map<Type, Integer> fromBasicReturnType = Map.ofEntries(
            Containers.concatArray(commonMap, returnMap)
    );

    public static final Map<Type, Integer> fromBasicFieldType = Map.ofEntries(
            Containers.concatArray(commonMap, fieldMap)
    );

    public static boolean isValidEventParameterType(Type type) {
        return fromBasicParamType.containsKey(type);
    }
}
